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
	"testing"

	A "github.com/IBM/fp-go/array"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceNameDataDisks = "onprem-datadisks"
)

var (
	sampleSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "sample"},
	}

	configRes = RefConfigMaps(&sampleSelector)
	secretRes = RefSecrets(&sampleSelector)
	diskRes   = RefResource("hpse.ibm.com/v1", ResourceNameDataDisks, &sampleSelector)

	noConfigRes = RefConfigMaps(nil)
	noSecretRes = RefSecrets(nil)
	noDiskRes   = RefResource("hpse.ibm.com/v1", ResourceNameDataDisks, nil)
)

func assertValidResourceRule(t *testing.T) func(rule *RelatedResourceRule) bool {
	return func(rule *RelatedResourceRule) bool {
		return assert.NotNil(t, rule, "rule must not be nil") && assert.NotNil(t, rule.MatchLabels, "label must not be nil")
	}
}

func assertValidResourceRules(t *testing.T) func(rule []*RelatedResourceRule) bool {
	assertRule := assertValidResourceRule(t)
	return A.Reduce(func(status bool, rule *RelatedResourceRule) bool {
		return status && assertRule(rule)
	}, true)
}

func TestNoRelatedResources(t *testing.T) {
	res := CreateRelatedResourceRules(A.Empty[RelatedResource]())
	assert.Empty(t, res)

	assertValidResourceRules(t)(res)
}

func TestAllRulesValid(t *testing.T) {
	res := CreateRelatedResourceRules([]RelatedResource{
		configRes,
		secretRes,
		diskRes,
	})
	// validate the size
	assert.Len(t, res, 3)
	// validate the content
	assertValidResourceRules(t)(res)
}

func TestAllRulesInvalid(t *testing.T) {
	res := CreateRelatedResourceRules([]RelatedResource{
		noConfigRes,
		noSecretRes,
		noDiskRes,
	})
	// validate the size
	assert.Empty(t, res)
	// validate the content
	assertValidResourceRules(t)(res)
}

func TestSomeRulesValid1(t *testing.T) {
	res := CreateRelatedResourceRules([]RelatedResource{
		configRes,
		noSecretRes,
		noDiskRes,
	})
	// validate the size
	assert.Len(t, res, 1)
	// validate the content
	assertValidResourceRules(t)(res)
}

func TestSomeRulesValid2(t *testing.T) {
	res := CreateRelatedResourceRules([]RelatedResource{
		configRes,
		secretRes,
		noDiskRes,
	})
	// validate the size
	assert.Len(t, res, 2)
	// validate the content
	assertValidResourceRules(t)(res)
}
