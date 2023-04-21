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

package datadiskref

import (
	"log"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	"libvirt.org/go/libvirtxml"
)

// createDataDiskRefReadyAction create the action
func createDataDiskRefReadyAction(disk *libvirtxml.StorageVolume) common.Action {

	return func() (*common.ResourceStatus, error) {
		// metadata to attach
		metadata := C.RawMap{
			"Name": disk.Name,
		}
		// marshal the disk info into metadata
		diskStrg, err := onprem.XMLMarshall(disk)
		if err == nil {
			metadata["diskXML"] = diskStrg
		} else {
			log.Printf("Unable to marshal the disk XML, cause: [%v]", err)
		}
		return &common.ResourceStatus{
			Status:      common.Ready,
			Description: diskStrg,
			Error:       nil,
			Metadata:    metadata,
		}, nil
	}
}

// CreateSyncAction synchronizes the state of the resource and determines what to do next
func CreateSyncAction(client *onprem.LivirtClient, opt *onprem.DataDiskRefOptions) common.Action {
	// checks for the validity of the data disk
	getDataDiskRef := onprem.GetDataDiskRef(client)
	diskXML, err := getDataDiskRef(opt)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	// ready
	return createDataDiskRefReadyAction(diskXML)
}
