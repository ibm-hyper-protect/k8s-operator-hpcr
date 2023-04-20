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

package common

import (
	C "github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	A "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/array"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	T "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/tuple"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	// version, name selector
	RelatedResource = T.Tuple3[string, string, *metav1.LabelSelector]
)

var (
	// RefResource creates a tuple describing a related resource
	RefResource = T.MakeTuple3[string, string, *metav1.LabelSelector]

	// CreateRelatedResourceRules constructs an array of related resources and filter out thoe that do not have a selector
	CreateRelatedResourceRules = F.Flow2(
		A.Filter(isValidRelatedResource),
		A.Map(createRelatedResourceRule),
	)
)

// createRelatedResourceRule constructs a resource rule from name and selector
func createRelatedResourceRule(res RelatedResource) *RelatedResourceRule {
	return &RelatedResourceRule{
		ResourceRule: ResourceRule{
			APIVersion: res.F1,
			Resource:   res.F2,
		},
		// select config maps by label
		LabelSelector: res.F3,
	}
}

// isValidRelatedResource tests if a resource is valid
func isValidRelatedResource(res RelatedResource) bool {
	return F.IsNonNil(res.F3)
}

// RefConfigMaps references a config map as related resource
func RefConfigMaps(labels *metav1.LabelSelector) RelatedResource {
	return RefResource(C.K8SAPIVersion, string(v1.ResourceConfigMaps), labels)
}

// RefSecrets references a secret as related resource
func RefSecrets(labels *metav1.LabelSelector) RelatedResource {
	return RefResource(C.K8SAPIVersion, string(v1.ResourceSecrets), labels)
}
