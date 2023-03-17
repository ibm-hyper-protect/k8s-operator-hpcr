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
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

// EnvFromConfigMaps merges all config maps into one
func EnvFromConfigMaps(data map[string]any) env.Environment {
	res := make(env.Environment)
	if related, ok := data["related"].(map[string]any); ok {
		// all config maps
		if configmaps, ok := related["ConfigMap.v1"].(map[string]any); ok {
			// iterate over all config maps and merge
			for _, item := range configmaps {
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
	}
	return res
}
