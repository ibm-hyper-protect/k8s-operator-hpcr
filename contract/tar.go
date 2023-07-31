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
	"bytes"
	"encoding/base64"

	E "github.com/IBM/fp-go/either"
	F "github.com/IBM/fp-go/function"
	R "github.com/IBM/fp-go/record"
	"github.com/ibm-hyper-protect/terraform-provider-hpcr/archive"
	"github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
)

// tarFolder tars the input folder and returns the result as a byte array
func tarFolder(folder string) E.Either[error, []byte] {
	return F.Pipe2(
		new(bytes.Buffer),
		archive.TarFolder[*bytes.Buffer](folder),
		E.Map[error]((*bytes.Buffer).Bytes),
	)
}

// upsertComposeArchive inserts the compose blob into the workload section
func upsertComposeArchive(data []byte) func(ctr contract.RawMap) contract.RawMap {
	// the new entry
	upsertEntry := R.UpsertAt[string, any]("compose", contract.RawMap{
		"archive": base64.StdEncoding.EncodeToString(data),
	},
	)
	// construct the upsert
	return func(ctr contract.RawMap) contract.RawMap {
		// workload section
		workload, ok := ctr[contract.KeyWorkload].(contract.RawMap)
		if !ok {
			workload = contract.RawMap{}
		}
		// add this top level
		return R.UpsertAt[string, any](contract.KeyWorkload, upsertEntry(workload))(ctr)
	}
}
