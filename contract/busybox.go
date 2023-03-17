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
	"path/filepath"

	"github.com/iancoleman/strcase"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	O "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/option"
	R "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/record"

	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
)

var (
	samplesRoot = "../samples"
	registry    = "docker-eu-public.artifactory.swg-devops.com"
	rootKey     = strcase.ToScreamingSnake(registry)

	lookupUsername = R.Lookup[string, string](fmt.Sprintf("%s_USERNAME", rootKey))
	lookupPassword = R.Lookup[string, string](fmt.Sprintf("%s_PASSWORD", rootKey))

	lookupIngestionHost = R.Lookup[string, string]("LOGDNA_INGESTION_HOST")
	lookupIngestionKey  = R.Lookup[string, string]("LOGDNA_INGESTION_KEY")
)

const (
	contractTemplate = `---
workload: 
  type: workload
  compose:
    archive: empty
env:
  type: env
  logging:
    logDNA:
      ingestionKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      hostname: TBD
`
)

// CreateContract constructs the contract for a specified image
func CreateContract(env map[string]string) func(composeFolder string) E.Either[error, C.RawMap] {
	// try to resolve loging
	ingestionHost := lookupIngestionHost(env)
	ingestionKey := lookupIngestionKey(env)

	// try to resolve credentials
	username := lookupUsername(env)
	password := lookupPassword(env)

	// construct pull secrets
	credentials := F.Pipe1(
		O.MonadSequence2(username, password, func(user, pwd string) O.Option[Credentials] {
			return O.Of(Credentials{
				registry: {Username: user, Password: pwd},
			})
		}),
		E.FromOption[error, Credentials](func() error { return fmt.Errorf("unable to lookup credentials") }),
	)

	// construct logging part
	logging := F.Pipe1(
		O.MonadSequence2(ingestionHost, ingestionKey, func(host, key string) O.Option[Logging] {
			return O.Of(Logging{
				LogDNA: &LogDNA{IngestionKey: key, Hostname: host},
			})
		}),
		E.FromOption[error, Logging](func() error { return fmt.Errorf("unable to lookup logging config") }),
	)

	// load the contract
	contract := F.Pipe1(
		C.ParseRawMapE([]byte(contractTemplate)),
		C.MapDerefRawMapE,
	)

	// create pull secrets
	contractWithAuth := F.Pipe2(
		credentials,
		E.Map[error](upsertPullSecrets),
		E.Ap[error, C.RawMap, C.RawMap](contract),
	)

	// create logging
	contractWithLogging := F.Pipe2(
		logging,
		E.Map[error](upsertLogging),
		E.Ap[error, C.RawMap, C.RawMap](contractWithAuth),
	)

	return F.Flow3(
		tarFolder,
		E.Map[error](upsertComposeArchive),
		E.Ap[error, C.RawMap, C.RawMap](contractWithLogging),
	)
}

// CreateBusyboxContract constructs the contract for the busybox image
func CreateBusyboxContract(env map[string]string) E.Either[error, C.RawMap] {
	return CreateContract(env)(filepath.Join(samplesRoot, "busybox"))
}
