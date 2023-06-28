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
	"context"
	"fmt"
	"log"
	"time"
)

// PanicAfterTimeout starts a timer and after a timeout panics and kills the process
// the timer can be cancelled via the cancel function, e.g. in a defer call
func PanicAfterTimeout(message string, delay time.Duration) context.CancelFunc {
	// manage the timeout
	timer := time.AfterFunc(delay, func() {
		log.Printf("[%s] did not complete after [%s], killing the process now ...", message, delay)
		panic(fmt.Sprintf("Timeout for [%s]", message))
	})
	return func() {
		timer.Stop()
	}
}
