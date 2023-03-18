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
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
)

// CreateSyncAction synchronizes the state of the resource and determines what to do next
func CreateSyncAction(client *onprem.LivirtClient, opt *onprem.DataDiskOptions) common.Action {
	// checks for the validity of the data disk
	isDataDiskValid := onprem.IsDataDiskValid(client)
	_, ok := isDataDiskValid(opt)
	if ok {
		// ready
		return common.CreateReadyAction()
	}
	// create a disk (will resize if required)
	diskSync := onprem.CreateDataDiskSync(client)
	_, err := diskSync(opt)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	// ready
	return common.CreateReadyAction()
}

func CreateFinalizeAction(client *onprem.LivirtClient, opt *onprem.DataDiskOptions) common.Action {
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
