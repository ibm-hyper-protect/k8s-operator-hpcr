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

package networkref

import (
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
)

// networkRefOptionsFromConfigMap decodes the information required to create a network ref
// from the k8s resource
func networkRefOptionsFromConfigMap(data *NetworkRefConfigResource, envMap env.Environment) (*onprem.NetworkRefOptions, error) {
	return &onprem.NetworkRefOptions{
		Name: onprem.BoxNetworkName(data.Parent.Spec.NetworkName),
	}, nil
}
