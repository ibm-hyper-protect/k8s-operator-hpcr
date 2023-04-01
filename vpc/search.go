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

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

const (
	KeyIBMCloudGsApiEndpoint     = "IBMCLOUD_GS_API_ENDPOINT"
	DefaultIBMCloudGsApiEndpoint = globalsearchv2.DefaultServiceURL
)

// GetGlobalSearchEndpoint returns the configured endpoint URL for the search API
func GetGlobalSearchEndpoint(env E.Environment) string {
	endpoint, ok := env[KeyIBMCloudGsApiEndpoint]
	if ok {
		return endpoint
	}
	return DefaultIBMCloudGsApiEndpoint
}

func CreateGlobalSearchService(auth core.Authenticator, globalSearchEndpoint string) (*globalsearchv2.GlobalSearchV2, error) {
	globalSearchService, err := globalsearchv2.NewGlobalSearchV2(&globalsearchv2.GlobalSearchV2Options{
		Authenticator: auth,
		URL:           globalSearchEndpoint,
	})
	if err != nil {
		log.Printf("Unable to create global search service, cause [%v]", err)
		return nil, err
	}
	return globalSearchService, nil
}

func CreateGlobalSearchServiceFromEnv(auth core.Authenticator, env E.Environment) (*globalsearchv2.GlobalSearchV2, error) {
	// create the service
	return CreateGlobalSearchService(auth, GetGlobalSearchEndpoint(env))
}
