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

package common

import (
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
)

type result[A any] struct {
	a A
	e error
}

// EromEither converts from Either to a normal tuple
func FromEither[A any](value E.Either[error, A]) (A, error) {
	res := E.Fold(func(e error) *result[A] {
		return &result[A]{e: e}
	}, func(a A) *result[A] {
		return &result[A]{a: a}
	})(value)
	return res.a, res.e
}
