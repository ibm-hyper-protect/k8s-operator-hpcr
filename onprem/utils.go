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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

const (
	// WaitSleepInterval time.
	waitSleepInterval = 1 * time.Second

	// WaitTimeout time.
	waitTimeout = 5 * time.Minute
)

// waitForSuccess wait for success and timeout after 5 minutes.
func waitForSuccess(errorMessage string, f func() error) error {
	start := time.Now()
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("[DEBUG] %s. Re-trying.\n", err)

		time.Sleep(waitSleepInterval)
		if time.Since(start) > waitTimeout {
			return fmt.Errorf("%s: %w", errorMessage, err)
		}
	}
}

func parseStorageVolumeXML(s string) (*libvirtxml.StorageVolume, error) {
	var volumeDef libvirtxml.StorageVolume
	err := xml.Unmarshal([]byte(s), &volumeDef)
	if err != nil {
		return nil, err
	}
	return &volumeDef, nil
}

func timeFromEpoch(str string) time.Time {
	var s, ns int

	ts := strings.Split(str, ".")
	if len(ts) == 2 {
		ns, _ = strconv.Atoi(ts[1])
	}
	s, _ = strconv.Atoi(ts[0])

	return time.Unix(int64(s), int64(ns))
}

func refreshPool(conn *libvirt.Libvirt) func(pool libvirt.StoragePool) error {
	return func(pool libvirt.StoragePool) error {
		return waitForSuccess("error refreshing pool for volume", func() error {
			log.Println("refreshing ...")
			return conn.StoragePoolRefresh(pool, 0)
		})
	}
}

type readerWithLog struct {
	rdr     io.Reader
	total   uint64
	current uint64
	t0      time.Time
}

func (r *readerWithLog) Read(p []byte) (int, error) {
	n, err := r.rdr.Read(p)
	if err == nil && r.total > 0 {
		r.current += uint64(n)
		t1 := time.Now()
		rel := float64(r.current) / float64(r.total)
		dt := t1.Sub(r.t0).Seconds()
		remaining := dt/rel - dt

		log.Printf("Read [%d bytes] from a total of [%d bytes], [%d %%], time remaining [%d s].", r.current, r.total, int(rel*100.0), int(remaining))
	}
	return n, err
}

func createReaderWithLog(rdr io.Reader, total uint64) io.Reader {
	return &readerWithLog{rdr: rdr, total: total, current: 0, t0: time.Now()}
}

func isError(err error, errorCode libvirt.ErrorNumber) bool {
	var perr libvirt.Error
	if errors.As(err, &perr) {
		return perr.Code == uint32(errorCode)
	}
	return false
}

func safeClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("Error during close: %v", err)
	}
}
