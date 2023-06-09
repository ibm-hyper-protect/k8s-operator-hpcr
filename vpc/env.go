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

package vpc

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

func EnvFromDotEnv(root string) (env.Environment, error) {
	f, err := os.Open(filepath.Clean(filepath.Join(root, ".env")))
	if err != nil {
		return nil, err
	}
	defer f.Close() // #nosec
	scanner := bufio.NewScanner(f)
	res := make(env.Environment)
	for scanner.Scan() {
		key, value, ok := env.SplitLine(scanner.Text())
		if ok {
			res[key] = value
		}
	}
	return res, nil
}
