// Copyright 2023 IBM Corp.
//
// Licensed under the Ache License, Version 2.0 (the "License");
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

package onprem

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// CreateMachineIdFromUUID creates a machineID from a UUID
func CreateMachineIdFromUUID(uid uuid.UUID) string {
	return hex.EncodeToString(uid[0:16])
}

// CreateMachineIdFromHash creates a hash of the input and then a machine ID
func CreateMachineIdFromHash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	// convert to a machine ID
	return hex.EncodeToString(h.Sum(nil)[0:16])
}

// CreateMachineIdFromMaybeUUID tries to parse the UUID from a string, then generate a machine ID from it
// if the string could not be decoded, create a machine ID from a hash instead
func CreateMachineIdFromMaybeUUID(maybeuuid string) string {
	// try to parse the uuid and use it if possible
	uid, err := uuid.Parse(maybeuuid)
	if err != nil {
		// fallback to a hash
		return CreateMachineIdFromHash(maybeuuid)
	}
	// from UUID
	return CreateMachineIdFromUUID(uid)
}
