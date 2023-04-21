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
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
)

// dataDiskRefOptionsFromConfigMap decodes the information required to create a data disk
// from the k8s resource
func dataDiskRefOptionsFromConfigMap(data *DataDiskRefConfigResource, envMap env.Environment) (*onprem.DataDiskRefOptions, error) {
	spec := data.Parent.Spec
	return &onprem.DataDiskRefOptions{
		Name:        spec.VolumeName,
		StoragePool: onprem.BoxStoragePool(spec.StoragePool),
	}, nil
}
