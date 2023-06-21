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

package datadisk

import (
	"log"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	"libvirt.org/go/libvirtxml"
)

// createDataDiskReadyAction create the action
func createDataDiskReadyAction(disk *libvirtxml.StorageVolume) (*common.ResourceStatus, error) {

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

// CreateSyncAction synchronizes the state of the resource and determines what to do next
func CreateSyncAction(client *onprem.LivirtClient, opt *onprem.DataDiskOptions) (*common.ResourceStatus, error) {
	// checks for the validity of the data disk
	isDataDiskValid := onprem.IsDataDiskValid(client)
	diskXML, ok := isDataDiskValid(opt)
	if ok {
		// ready
		return createDataDiskReadyAction(diskXML)
	}
	// create a disk (will resize if required)
	diskSync := onprem.CreateDataDiskSync(client)
	disk, err := diskSync(opt)
	if err != nil {
		log.Printf("Unable to create data disk [%s], cause: [%v]", opt.Name, err)
		return common.CreateErrorAction(err)
	}
	// try to get the XML description
	getDiskXML := onprem.GetStorageVolXMLDesc(client)
	diskXML, err = getDiskXML(disk)
	if err != nil {
		log.Printf("Unable to get disk XML [%s], cause: [%v]", opt.Name, err)
		return common.CreateErrorAction(err)
	}
	// ready
	return createDataDiskReadyAction(diskXML)
}

func CreateFinalizeAction(client *onprem.LivirtClient, opt *onprem.DataDiskOptions) (*common.ResourceStatus, error) {
	// TODO proper check for existence comes here
	// ...
	// destroy the instance
	deleteSync := onprem.DeleteDataDiskSync(client)
	err := deleteSync(opt.StoragePool, opt.Name)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	// done
	return common.CreateReadyAction()
}
