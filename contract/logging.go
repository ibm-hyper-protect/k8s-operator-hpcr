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

package contract

import (
	R "github.com/IBM/fp-go/record"
	"github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
)

type LogDNA struct {
	IngestionKey string `json:"ingestionKey" yaml:"ingestionKey"`
	Hostname     string `json:"hostname" yaml:"hostname"`
}

type Logging struct {
	LogDNA *LogDNA `json:"logDNA" yaml:"logDNA"`
}

// upsertComposeArchive inserts the logging config into the environment section of a contract
func upsertLogging(logging Logging) func(ctr contract.RawMap) contract.RawMap {
	// the new entry
	upsertLog := R.UpsertAt[string, any]("logging", logging)
	// construct the upsert
	return func(ctr contract.RawMap) contract.RawMap {
		// env section
		env, ok := ctr[contract.KeyEnv].(contract.RawMap)
		if !ok {
			env = contract.RawMap{}
		}
		// add this top level
		return R.UpsertAt[string, any](contract.KeyEnv, upsertLog(env))(ctr)
	}
}
