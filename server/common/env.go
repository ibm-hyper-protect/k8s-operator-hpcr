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
	"encoding/base64"
	"fmt"
	"log"

	C "github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

var (
	keyConfigMap = fmt.Sprintf("%s.%s", "ConfigMap", C.K8SAPIVersion)
	keySecret    = fmt.Sprintf("%s.%s", "Secret", C.K8SAPIVersion)
)

// EnvFromConfigMapsOrSecrets merges all config maps into one
func EnvFromConfigMapsOrSecrets(data map[string]any) env.Environment {
	res := make(env.Environment)
	if related, ok := data["related"].(map[string]any); ok {
		// all config maps
		if configmaps, ok := related[keyConfigMap].(map[string]any); ok {
			// iterate over all config maps and merge
			for name, item := range configmaps {
				log.Printf("Merging ConfigMap [%s] ...", name)
				if configmap, ok := item.(map[string]any); ok {
					// extract data
					if configmapdata, ok := configmap["data"].(map[string]any); ok {
						// merge
						for key, value := range configmapdata {
							if strgVal, ok := value.(string); ok {
								res[key] = strgVal
							}
						}
					}
				}
			}
		}
		// all secrets maps
		if secrets, ok := related[keySecret].(map[string]any); ok {
			// iterate over all config maps and merge
			for name, item := range secrets {
				log.Printf("Merging Secret [%s] ...", name)
				if secret, ok := item.(map[string]any); ok {
					// extract data
					if secretdata, ok := secret["data"].(map[string]any); ok {
						// merge
						for key, value := range secretdata {
							if strgVal, ok := value.(string); ok {
								// secrets are baes64 encoded
								decValue, err := base64.StdEncoding.DecodeString(strgVal)
								if err == nil {
									res[key] = string(decValue)
								} else {
									log.Printf("Unable to base64 decode the secret [%s], cause: [%v]", name, err)
								}
							}
						}
					}
				}
			}
		}
	}
	return res
}
