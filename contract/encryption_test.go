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
	"testing"

	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	B "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/bytes"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestSignAndEncryptContract(t *testing.T) {
	// load env
	env, err := godotenv.Read("../.env")
	if err != nil {
		t.SkipNow()
	}

	// load the encryption certificate and create the encryption callback
	enc := F.Pipe2(
		env,
		LoadPublicKeyFromEnv,
		E.Map[error](EncryptContract),
	)

	ctr := F.Pipe2(
		env,
		CreateBusyboxContract,
		E.Chain(ValidateContract),
	)

	encCtr := F.Pipe5(
		enc,
		E.Ap[error, C.RawMap, E.Either[error, C.RawMap]](ctr),
		E.Flatten[error, C.RawMap],
		C.MapRefRawMapE,
		E.Chain(C.StringifyRawMapE),
		E.Map[error](B.ToString),
	)

	assert.True(t, E.IsRight(encCtr))
}
