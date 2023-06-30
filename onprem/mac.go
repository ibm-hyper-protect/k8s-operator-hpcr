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
	"fmt"

	"github.com/google/uuid"
)

// createMacAddressFromBytes creates a local, unicast mac address
func createMacAddressFromBytes(data [6]byte) string {
	// lsb, make sure to identify as a local address and set the multicast bit to zero (see https://en.wikipedia.org/wiki/MAC_address)
	lsb := (data[0] & 0xfe) | 0x02
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", lsb, data[1], data[2], data[3], data[4], data[5])
}

// CreateMacAddressFromUUID creates a mac address from the first 6 bytes of the UUID
func CreateMacAddressFromUUID(uid uuid.UUID) string {
	// need 6 bytes for the address
	mac := [6]byte{}
	copy(mac[:], uid[:])
	// produce a valid mac address
	return createMacAddressFromBytes(mac)
}

// constructs a hash value for the string and produces a mac address from that
func CreateMacAddressFromHash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	// need 6 bytes for the address
	mac := [6]byte{}
	copy(mac[:], h.Sum(nil))
	// produce a valid mac address
	return createMacAddressFromBytes(mac)
}

// CreateMacAddressFromMaybeUUID tries to parse the UUID from a string, then generate a MAC address from it
// if the string could not be decoded, create a MAC from a hash instead
func CreateMacAddressFromMaybeUUID(maybeuuid string) string {
	// try to parse the uuid and use it if possible
	uid, err := uuid.Parse(maybeuuid)
	if err != nil {
		// fallback to a hash
		return CreateMacAddressFromHash(maybeuuid)
	}
	// from UUID
	return CreateMacAddressFromUUID(uid)
}
