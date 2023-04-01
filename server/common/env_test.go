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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/vpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readJson(name string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join("..", "..", "samples", name))
	if err != nil {
		return nil, err
	}
	var res map[string]any
	err = json.Unmarshal(data, &res)
	return res, err
}

func TestEnvFromConfigMaps1(t *testing.T) {
	data, err := readJson("create_resource.json")
	require.NoError(t, err)

	env := EnvFromConfigMapsOrSecrets(data)
	assert.NotNil(t, env["IBMCLOUD_IS_API_ENDPOINT"])
}

func TestEnvFromConfigMaps2(t *testing.T) {
	data, err := readJson("create_resource_full.json")
	require.NoError(t, err)

	env := EnvFromConfigMapsOrSecrets(data)

	apiKey, err := vpc.GetIBMCloudApiKey(env)
	require.NoError(t, err)

	region := vpc.GetRegion(env)
	defEndpoint := vpc.GetDefaultIBMCloudApiEndpoint(region)
	endpoint := vpc.GetIBMCloudApiEndpoint(env, defEndpoint)
	iamEndpoint := vpc.GetIBMCloudIAMApiEndpoint(env)

	assert.Equal(t, "xxx", apiKey)
	assert.Equal(t, "https://us-south-stage01.iaasdev.cloud.ibm.com", endpoint)
	assert.Equal(t, "https://iam.test.cloud.ibm.com", iamEndpoint)
}
