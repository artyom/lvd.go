// Copyright 2012 Luuk van Dijk. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// read sample NetCDF file from a directory default ./testdata/, or environment variable
// NETCDF_TESTDIR if set, read, copy over through the API and verify that the written file
// is byte by byte identical.

package cdf

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestData(t *testing.T) {
//	testAllFiles(t, readWriteCompareData)
}

func readWriteCompareData(srcpath string, t *testing.T) {
	srcf, err := os.Open(srcpath)
	if err != nil {
		t.Error(err)
		return
	}

	src, err := Open(srcf)
	if err != nil {
		t.Error(err)
		return
	}

	if errs := src.Header.Check(); errs != nil {
		fmt.Println(src)
		t.Errorf("%v", errs)
		return
	}

	dstf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Error(err)
		return
	}
	//	log.Println("tmp file: ", dstf.Name())
	defer os.Remove(dstf.Name())

	dst, err := Create(dstf, src.Header)
	if err != nil {
		t.Error(err)
		return
	}

	for _, v := range dst.Header.Variables() {
		r := src.Reader(v, nil, nil)
		w := dst.Writer(v, nil, nil)
		buf := r.Zero(-1)
		for {
			nr, err := r.Read(buf)
			nw, erw := w.Write(buf)
			if nr == nw && err == nil && erw == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			t.Errorf("read: %v/%v write: %v/%v", nr, err, nw, erw)
			break
		}
	}

	UpdateNumRecs(dstf)

	// compare

	srcf.Seek(0, 0)
	dstf.Seek(0, 0)

	srcd, err := ioutil.ReadAll(srcf)
	dstd, err := ioutil.ReadAll(dstf)
	
	if len(srcd) != len(dstd) {
		t.Error(srcpath, ":different lengths", len(srcd), len(dstd))
	}

	for i := 0; i < len(srcd) && i < len(dstd); i++ {
		if srcd[i] != dstd[i] {
			t.Error(srcpath, ":difference at offset ", i)
			break
		}
	}
}
