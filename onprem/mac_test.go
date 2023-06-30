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

package onprem

import (
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacAddress(t *testing.T) {

	mac := CreateMacAddressFromHash("This is some input")
	addr, err := net.ParseMAC(mac)

	require.NoError(t, err)
	assert.Equal(t, addr[:], net.HardwareAddr{0xb6, 0x8c, 0xca, 0xe7, 0x6f, 0xf1})
}

func TestMacAddressFromUUID(t *testing.T) {

	uid, err := uuid.Parse("d3414e67-a26f-4791-96f1-cd842c15346c")
	require.NoError(t, err)

	mac := CreateMacAddressFromUUID(uid)
	addr, err := net.ParseMAC(mac)

	require.NoError(t, err)
	assert.Equal(t, addr[:], net.HardwareAddr{0xd2, 0x41, 0x4e, 0x67, 0xa2, 0x6f})
}
