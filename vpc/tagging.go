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
	"log"

	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
)

const (
	KeyIBMCloudGtApiEndpoint     = "IBMCLOUD_GT_API_ENDPOINT"
	DefaultIBMCloudGtApiEndpoint = globaltaggingv1.DefaultServiceURL
)

func GetIBMCloudGtApiEndpoint(env E.Environment) string {
	endpoint, ok := env[KeyIBMCloudGtApiEndpoint]
	if ok {
		return endpoint
	}
	return DefaultIBMCloudGtApiEndpoint
}

func CreateGlobalTaggingService(auth core.Authenticator, globalTaggingEndpoint string) (*globaltaggingv1.GlobalTaggingV1, error) {
	globalSearchService, err := globaltaggingv1.NewGlobalTaggingV1(&globaltaggingv1.GlobalTaggingV1Options{
		Authenticator: auth,
		URL:           globalTaggingEndpoint,
	})
	if err != nil {
		log.Printf("Unable to create global tagging service, cause [%v]", err)
		return nil, err
	}
	return globalSearchService, nil
}

func CreateTaggingServiceFromEnv(auth core.Authenticator, env E.Environment) (*globaltaggingv1.GlobalTaggingV1, error) {
	return CreateGlobalTaggingService(auth, GetIBMCloudGtApiEndpoint(env))

}
