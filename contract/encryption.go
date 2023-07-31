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
	E "github.com/IBM/fp-go/either"
	F "github.com/IBM/fp-go/function"
	I "github.com/IBM/fp-go/identity"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	CE "github.com/ibm-hyper-protect/terraform-provider-hpcr/encrypt"
)

var (
	// defaultEncryption represents the encryption and signing callbacks we need
	// to encrypt and sign the contract. The object will automatically handle openssl or golang crypto support under the hoods
	defaultEncryption = CE.DefaultEncryption()
)

// EncryptContract creates an encryption function on top of an encryption certificate that will encrypt
// and sign the contract. The signing key will be a temporary private key
func EncryptContract(encCert []byte) func(contract C.RawMap) E.Either[error, C.RawMap] {
	// encryptAndSign derives the encryptor based on the encryption certificate
	// the first parameter into the method is the private key used for signing
	encryptAndSign := C.EncryptAndSignContract(defaultEncryption.EncryptBasic(encCert), defaultEncryption.SignDigest, defaultEncryption.PubKey)
	return func(contract C.RawMap) E.Either[error, C.RawMap] {
		// create a temporary private key for signing
		return F.Pipe1(
			defaultEncryption.PrivKey(),
			E.Chain(F.Flow2(
				encryptAndSign,
				I.Ap[E.Either[error, C.RawMap]](contract),
			)),
		)
	}
}
