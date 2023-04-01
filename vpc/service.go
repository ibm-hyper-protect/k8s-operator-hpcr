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
	"log"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"

	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

func CreateVpcService(auth core.Authenticator, isApiEndpoint string) (*vpcv1.VpcV1, error) {
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: auth,
		URL:           fmt.Sprintf("%s/v1", isApiEndpoint),
	})
	if err != nil {
		log.Printf("Unable to create VPC Service, cause [%v]", err)
		return nil, err
	}
	return vpcService, nil
}

func CreateVpcServiceFromEnvAndRegion(auth core.Authenticator, region string, env E.Environment) (*vpcv1.VpcV1, error) {
	// some logging
	log.Printf("Getting VPC service for region [%s] ...", region)
	// locate the endpoint
	defEndpoint := GetDefaultIBMCloudApiEndpoint(region)
	endpoint := GetIBMCloudApiEndpoint(env, defEndpoint)

	// create the service
	return CreateVpcService(auth, endpoint)
}

func CreateVpcServiceFromEnv(auth core.Authenticator, env E.Environment) (*vpcv1.VpcV1, error) {
	return CreateVpcServiceFromEnvAndRegion(auth, GetRegion(env), env)
}
