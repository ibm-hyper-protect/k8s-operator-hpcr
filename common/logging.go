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
	"log"
	"sync/atomic"
	"time"
)

var count uint64

// EntryExit immediately prints an entry statement and returns a function that can be used with defer and that prints an exit statement with timing
func EntryExit(method string) func() {
	tEnter := time.Now()
	idx := atomic.AddUint64(&count, 1)
	log.Printf("Enter [%d]: %s", idx, method)

	return func() {
		tExit := time.Now()
		log.Printf("Exit  [%d]: %s in [%s]", idx, method, tExit.Sub(tEnter))
	}
}
