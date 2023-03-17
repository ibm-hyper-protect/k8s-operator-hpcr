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
	"context"
	"os"

	ENC "github.com/ibm-hyper-protect/terraform-provider-hpcr/encrypt"
	A "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/array"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	O "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/option"
	"github.com/qri-io/jsonschema"

	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	V "github.com/ibm-hyper-protect/terraform-provider-hpcr/validation"
)

var (
	readFile = E.Eitherize1(os.ReadFile)
	// LoadRawContractFromYAML reads a contract from a YAML file
	LoadRawContractFromYAML = F.Flow3(
		readFile,
		E.Chain(C.ParseRawMapE),
		C.MapDerefRawMapE,
	)
	// the contract schema
	contractSchema = V.GetContractSchema()
)

// DefaultEncryption returns the default encryption implementation
func DefaultEncryption() ENC.Encryption {
	return ENC.DefaultEncryption()
}

func validate[A any](schema *jsonschema.Schema) func(A) []jsonschema.KeyError {
	return func(data A) []jsonschema.KeyError {
		return *schema.Validate(context.Background(), data).Errs
	}
}

// ValidateContract validates a contract against its JSON schema
func ValidateContract(contract C.RawMap) E.Either[error, C.RawMap] {
	return F.Pipe3(
		contractSchema,
		E.Map[error](validate[C.RawMap]),
		E.Ap[error, C.RawMap, []jsonschema.KeyError](E.Of[error](contract)),
		E.Chain(F.Flow3(
			A.Head[jsonschema.KeyError],
			O.Map(func(err jsonschema.KeyError) error { return err }),
			O.Fold(F.Constant(E.Of[error](contract)), E.Left[error, C.RawMap]),
		)),
	)
}
