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
	"fmt"
	"testing"

	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	B "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/bytes"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	"github.com/joho/godotenv"
)

func TestSimpleContract(t *testing.T) {
	// load env
	env, err := godotenv.Read("../.env")
	if err != nil {
		t.SkipNow()
	}
	// build

	ctr := F.Pipe5(
		env,
		CreateBusyboxContract,
		E.Chain(ValidateContract),
		C.MapRefRawMapE,
		E.Chain(C.StringifyRawMapE),
		E.Map[error](B.ToString),
	)

	fmt.Println(ctr)
}
