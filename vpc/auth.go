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

package vpc

import (
	"fmt"

	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

const (
	DefaultIBMCloudIAMApiEndpoint = "https://iam.cloud.ibm.com"
	KeyIBMCloudIAMApiEndpoint     = "IBMCLOUD_IAM_API_ENDPOINT"
	KeyIBMCloudApiKey             = "IBMCLOUD_API_KEY" // #nosec
)

func GetIBMCloudIAMApiEndpoint(env E.Environment, defEndpoint string) string {
	endpoint, ok := env[KeyIBMCloudIAMApiEndpoint]
	if ok {
		return endpoint
	}
	return defEndpoint
}

func GetIBMCloudApiKey(env E.Environment) (string, error) {
	apiKey, ok := env[KeyIBMCloudApiKey]
	if ok {
		return apiKey, nil
	}
	return apiKey, fmt.Errorf("API Key [%s] not found", KeyIBMCloudApiKey)
}
