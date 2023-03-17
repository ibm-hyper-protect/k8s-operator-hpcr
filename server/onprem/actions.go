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

package onprem

import (
	"log"
	"strings"

	"github.com/ibm-hyper-protect/hpcr-controller/onprem"
	"github.com/ibm-hyper-protect/hpcr-controller/server/common"
	A "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/array"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	"libvirt.org/go/libvirtxml"
)

func createInstanceRunningAction(client *onprem.LivirtClient, inst *libvirtxml.Domain, opt *onprem.InstanceOptions) common.Action {

	getLoggingVolume := onprem.GetLoggingVolume(client)

	return func() (*common.ResourceStatus, error) {
		// fetch the logs
		log.Printf("Domain [%s] is running, fetching logs ...", opt.Name)
		// try to get the content of the logging volume
		logName := onprem.GetLoggingVolumeName(opt.Name)
		data, err := getLoggingVolume(opt.StoragePool, logName)
		if err != nil {
			// returns some error status
			return &common.ResourceStatus{
				Status:      common.Waiting,
				Description: err.Error(),
				Error:       err,
			}, err
		}
		// parse log into lines
		lines := F.Pipe1(
			strings.Split(data, "\n"),
			A.Map(strings.TrimSpace),
		)
		// partition the lines
		success, failure := onprem.PartitionLogs(lines)
		if onprem.VSIFailedToStart(failure) {
			// print some error details
			desc := strings.Join(failure, "\n")
			log.Printf("Domain [%s] failed to start, errors: [%s]", opt.Name, desc)
			// VSI is ready but in an error state. It won't start at the next attempt
			return &common.ResourceStatus{
				Status:      common.Ready,
				Description: desc,
				Error:       nil,
			}, nil
		}
		// check if we are still booting
		if onprem.VSIStartedSuccessfully(success) {
			// juhuuu
			return &common.ResourceStatus{
				Status:      common.Ready,
				Description: strings.Join(lines, "\n"),
				Error:       nil,
			}, nil
		}
		// log this
		desc := strings.Join(lines, "\n")
		log.Printf("Domain [%s] is still booting, logs: [%s]", opt.Name, desc)
		// we need to wait
		return &common.ResourceStatus{
			Status:      common.Waiting,
			Description: desc,
			Error:       nil,
		}, nil
	}
}

// CreateSyncAction synchronizes the state of the resource and determines what to do next
func CreateSyncAction(client *onprem.LivirtClient, opt *onprem.InstanceOptions) common.Action {
	// checks for the validity of the instance
	isInstanceValid := onprem.IsInstanceValid(client)
	inst, ok := isInstanceValid(opt)
	if ok {
		// validate the instance
		return createInstanceRunningAction(client, inst, opt)
	}
	// start the instance
	instSync := onprem.CreateInstanceSync(client)
	result, err := instSync(opt)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	// log the result
	resultStrg, err := onprem.XMLMarshall(result)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	log.Printf("Instance: %s", resultStrg)
	// we need an additional sync to tell if the instance is ready
	return common.CreateWaitingAction()
}

func CreateFinalizeAction(client *onprem.LivirtClient, opt *onprem.InstanceOptions) common.Action {
	// TODO proper check for existence comes here
	// ...
	// destroy the instance
	deleteSync := onprem.DeleteInstanceSync(client)
	err := deleteSync(opt.StoragePool, opt.Name)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	// done
	return common.CreateReadyAction()
}
