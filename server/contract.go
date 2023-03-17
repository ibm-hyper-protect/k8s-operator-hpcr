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

package server

import (
	"fmt"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

func CreateContract(data map[string]any, envMap env.Environment) (string, error) {
	if parent, ok := data["parent"].(map[string]any); ok {
		if spec, ok := parent["spec"].(map[string]any); ok {
			if contract, ok := spec["contract"].(string); ok {
				// this is the contract
				fmt.Printf("contract: %s", contract)
				return contract, nil
			}
		}
	}
	return "", fmt.Errorf("no contract field")
}
