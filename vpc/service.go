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

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"

	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

func CreateAuthenticator(apiKey string, isIAMApiEndpoint string) (*core.IamAuthenticator, error) {
	return &core.IamAuthenticator{
		ApiKey: apiKey,
		URL:    isIAMApiEndpoint,
	}, nil
}

func CreateVpcService(apiKey string, isApiEndpoint string, isIAMApiEndpoint string) (*vpcv1.VpcV1, error) {
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
			URL:    isIAMApiEndpoint,
		},
		URL: fmt.Sprintf("%s/v1", isApiEndpoint),
	})
	if err != nil {
		return nil, err
	}
	return vpcService, nil
}

func CreateVpcServiceFromEnv(env E.Environment) (*vpcv1.VpcV1, error) {
	apiKey, err := GetIBMCloudApiKey(env)
	if err != nil {
		return nil, err
	}
	region := GetRegion(env)
	defEndpoint := GetDefaultIBMCloudApiEndpoint(region)
	endpoint := GetIBMCloudApiEndpoint(env, defEndpoint)
	iamEndpoint := GetIBMCloudIAMApiEndpoint(env, DefaultIBMCloudIAMApiEndpoint)

	// create the service
	return CreateVpcService(apiKey, endpoint, iamEndpoint)
}

func CreateTaggingServiceFromEnv(env E.Environment) (*GlobalTagging, error) {
	apiKey, err := GetIBMCloudApiKey(env)
	if err != nil {
		return nil, err
	}
	endpoint := GetIBMCloudGtApiEndpoint(env)
	iamEndpoint := GetIBMCloudIAMApiEndpoint(env, DefaultIBMCloudIAMApiEndpoint)

	return CreateGlobalTagging(&GlobalTaggingOptions{
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
			URL:    iamEndpoint,
		},
		URL: fmt.Sprintf("%s/v3", endpoint),
	})

}
