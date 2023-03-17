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

package cli

import (
	"log"
	"strconv"
	"time"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server"
	c "github.com/urfave/cli/v2"
)

const (
	portFlagName = "port"
)

// version number, will be injected by the build
var (
	version  string
	compiled string
)

// CreateApp creates the application that starts the operator
func CreateApp() *c.App {
	compileTime, _ := strconv.ParseInt(compiled, 10, 64)
	compiledAt := time.UnixMilli(compileTime * 1000)
	return &c.App{
		Version:  version,
		Compiled: compiledAt,
		Name:     "hpcr-controller",
		Usage:    "Start HPCR controller",
		Flags: []c.Flag{
			&c.IntFlag{
				Name:    portFlagName,
				Aliases: []string{"p"},
				Value:   8080,
				Usage:   "Port to listen on",
			},
		},
		Action: func(ctx *c.Context) error {
			port := ctx.Int(portFlagName)

			log.Printf("Starting server [%s] built on [%v] on port [%d] ...", version, compiledAt, port)

			svr := server.CreateServer(version, compiled)

			return svr(port)
		},
	}
}
