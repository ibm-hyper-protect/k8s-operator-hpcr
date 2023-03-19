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
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
)

func boxSize(size uint64) uint64 {
	if size <= 0 {
		return onprem.DefaultDataDiskSize
	}
	return size
}

func boxStoragePool(pool string) string {
	if len(pool) <= 0 {
		return onprem.DefaultStoragePool
	}
	return pool
}

// dataDiskOptionsFromConfigMap decodes the information required to create a data disk
// from the k8s resource
func dataDiskOptionsFromConfigMap(data *DataDiskConfigResource, envMap env.Environment) (*onprem.DataDiskOptions, error) {
	spec := data.Parent.Spec
	opt := &onprem.DataDiskOptions{
		Name:        string(data.Parent.UID),
		StoragePool: boxStoragePool(spec.StoragePool),
		Size:        boxSize(spec.Size),
	}
	return opt, nil
}
