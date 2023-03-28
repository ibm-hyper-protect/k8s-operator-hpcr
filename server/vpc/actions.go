// Copyright 2023 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package datasource

package vpc

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/vpc"
)

var TagPrefix = strings.ReplaceAll(ServicePrefix, "-", "_")

func deleteInstanceAction(service *vpcv1.VpcV1, inst *vpcv1.Instance) common.Action {
	return func() (*common.ResourceStatus, error) {
		_, err := service.DeleteInstance(&vpcv1.DeleteInstanceOptions{ID: inst.ID})
		if err != nil {
			return common.CreateErrorAction(err)()
		}
		// log that we deleted the instance
		log.Printf("Deleted instance [%s]", *inst.ID)
		return common.CreateWaitingAction()()
	}
}

func createTag(data string) (string, error) {
	// construct sha256 over userdata
	h := sha256.New()
	_, err := h.Write([]byte(data))
	if err != nil {
		return "", err
	}
	bs := h.Sum(nil)
	return fmt.Sprintf("%s:%x", TagPrefix, bs), nil
}

func createInstanceAction(service *vpcv1.VpcV1, taggingSvc *globaltaggingv1.GlobalTaggingV1, vpcOp *vpcv1.CreateInstanceOptions, opt *InstanceOptions) common.Action {
	return func() (*common.ResourceStatus, error) {
		// construct instance
		inst, _, err := service.CreateInstance(vpcOp)
		if err != nil {
			return common.CreateErrorAction(err)()
		}
		// the tag
		tag, err := createTag(opt.UserData)
		if err != nil {
			return common.CreateErrorAction(err)()
		}
		// attach this service tag
		tagType := globaltaggingv1.AttachTagOptionsTagTypeUserConst
		res, _, err := taggingSvc.AttachTag(&globaltaggingv1.AttachTagOptions{
			Resources: []globaltaggingv1.Resource{{ResourceID: inst.CRN}},
			TagNames:  []string{tag},
			TagType:   &tagType,
		})
		if err != nil {
			return common.CreateErrorAction(err)()
		}
		// validate the response
		if len(res.Results) != 1 {
			return common.CreateErrorAction(err)()
		}
		// log that we created the instance
		log.Printf("Created instance [%s]", *inst.ID)
		return common.CreateWaitingAction()()
	}
}

func isString(msg, left, right string) bool {
	if left != right {
		log.Printf("Mismatch [%s] != [%s], [%s]", left, right, msg)
		return false
	}
	return true
}

func isSubnet(opt *InstanceOptions, inst *vpcv1.Instance) bool {
	if inst.PrimaryNetworkInterface != nil && inst.PrimaryNetworkInterface.Subnet != nil {
		return isString("subnet", opt.SubnetID, *inst.PrimaryNetworkInterface.Subnet.ID)
	}
	log.Printf("No subnet assigned to [%s]", *inst.ID)
	return false
}

func getTags(taggingSvc *globaltaggingv1.GlobalTaggingV1, inst *vpcv1.Instance) (*globaltaggingv1.TagList, error) {
	// read the attached tag
	tagType := globaltaggingv1.AttachTagOptionsTagTypeUserConst
	list, _, err := taggingSvc.ListTags(&globaltaggingv1.ListTagsOptions{TagType: &tagType, AttachedTo: inst.CRN})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func isTag(opt *InstanceOptions, inst *vpcv1.Instance, tags *globaltaggingv1.TagList) bool {
	// compute the tag for reference
	localTag, err := createTag(opt.UserData)
	if err != nil {
		log.Printf("Unable to create tag: %s", err.Error())
		return false
	}
	// check if the tags contain the desired one
	for _, tag := range tags.Items {
		if tag.Name != nil && *tag.Name == localTag {
			return true
		}
	}
	// error out
	log.Printf("Attached tags [%v] do not match [%s] for [%s].", tags, localTag, *inst.CRN)
	return false
}

func isVsiConfigValid(opt *InstanceOptions, inst *vpcv1.Instance, tags *globaltaggingv1.TagList) bool {
	// validate
	return isString("vpc", opt.VpcID, *inst.VPC.ID) &&
		isString("zone", opt.ZoneName, *inst.Zone.Name) &&
		isString("image", opt.ImageID, *inst.Image.ID) &&
		isString("profile", opt.ProfileName, *inst.Profile.Name) &&
		isSubnet(opt, inst) &&
		isTag(opt, inst, tags)
}

func createRunningInstanceAction(inst *vpcv1.Instance, opt *InstanceOptions) common.Action {
	return func() (*common.ResourceStatus, error) {
		// prepare some metadata
		metadata := make(map[string]any)
		instData, err := json.Marshal(inst)
		if err == nil {
			metadata["instance"] = string(instData)
		}
		// return the status
		return &common.ResourceStatus{
			Status:      common.Ready,
			Description: *inst.Name,
			Error:       nil,
			Metadata:    metadata,
		}, nil

	}
}

func CreateSyncAction(vpcSvc *vpcv1.VpcV1, taggingSvc *globaltaggingv1.GlobalTaggingV1, opt *InstanceOptions) common.Action {
	// check for the existence of the instance
	inst, err := vpc.FindInstance(vpcSvc, opt.Name)
	if err != nil {
		// if the instance was not found, create it
		if errors.Is(err, vpc.InstanceNotFound) {
			// log this
			log.Printf("The VSI [%s] could not be found, creating it ...", opt.Name)
			// construct the instance
			vpcOpt, err := CreateVpcInstanceOptions(opt)
			if err != nil {
				return common.CreateErrorAction(err)
			}
			return createInstanceAction(vpcSvc, taggingSvc, vpcOpt, opt)
		}
		// general error
		return common.CreateErrorAction(err)
	}
	// status
	status := *inst.Status
	log.Printf("The VSI [%s] is in status [%s]", *inst.ID, status)
	// check if the instance if in a valid state
	switch status {
	// wait until deleted, then retry to create later
	case vpcv1.InstanceStatusDeletingConst:
		return common.CreateStatusAction(common.Waiting)
	// delete the VSI
	case vpcv1.InstanceStatusFailedConst:
	case vpcv1.InstanceStatusRestartingConst:
	case vpcv1.InstanceStatusStoppedConst:
	case vpcv1.InstanceStatusStoppingConst:
		return deleteInstanceAction(vpcSvc, inst)
	// validate and wait if validation is successful
	case vpcv1.InstanceStatusPendingConst:
	case vpcv1.InstanceStatusStartingConst:
		tags, err := getTags(taggingSvc, inst)
		if err != nil {
			return common.CreateErrorAction(err)
		}
		if isVsiConfigValid(opt, inst, tags) {
			return common.CreateStatusAction(common.Waiting)
		}
		// if config is not ok, delete the instance
		return deleteInstanceAction(vpcSvc, inst)
	// validate and signal ready if validation is successful
	case vpcv1.InstanceStatusRunningConst:
		tags, err := getTags(taggingSvc, inst)
		if err != nil {
			return common.CreateErrorAction(err)
		}
		if isVsiConfigValid(opt, inst, tags) {
			return createRunningInstanceAction(inst, opt)
		}
		// if config is not ok, delete the instance
		return deleteInstanceAction(vpcSvc, inst)
	}
	// per default try to delete the VSI
	return deleteInstanceAction(vpcSvc, inst)
}

func CreateFinalizeAction(service *vpcv1.VpcV1, opt *InstanceOptions) common.Action {
	// check for the existence of the instance
	inst, err := vpc.FindInstance(service, opt.Name)
	if err != nil {
		// if the instance was not found, this is good
		if errors.Is(err, vpc.InstanceNotFound) {
			// success
			return common.CreateReadyAction()
		}
		// general error
		return common.CreateErrorAction(err)
	}
	// status
	status := *inst.Status
	log.Printf("The VSI [%s] is in status [%s]", *inst.ID, status)
	// check if the instance if in a valid state
	switch status {
	// wait until deleted, then retry to create later
	case vpcv1.InstanceStatusDeletingConst:
		return common.CreateStatusAction(common.Waiting)
	default:
		return deleteInstanceAction(service, inst)
	}
}
