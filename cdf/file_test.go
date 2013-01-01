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
	//"log"
	"os"
	"testing"
)

func TestData(t *testing.T) { testAllFiles(t, readWriteCompareData) }

func readWriteCompareData(srcpath string, t *testing.T) {
	srcf, err := os.Open(srcpath)
	if err != nil {
		t.Error(err)
		return
	}

	srcfi, err := srcf.Stat()
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
	//log.Println("tmp file: ", dstf.Name())
	defer os.Remove(dstf.Name())

	dst, err := Create(dstf, src.Header)
	if err != nil {
		t.Error(err)
		return
	}

	//log.Print(src.Header)
	numrecs := src.Header.NumRecs(srcfi.Size())

	//log.Print("filling ", src.Header.numrecs, " records")
	for i := 0; i < int(numrecs); i++ {
		if err := dst.FillRecord(i); err != nil {
			t.Error(err)
			break
		}
	}

	for i, v := range dst.Header.Variables() {

		if !dst.Header.IsRecordVariable(v) {
			//log.Print("filling ", v, "...")
			// TODO: only for dtype is BYTE or SHORT, save time
			if err := dst.Fill(v); err != nil {
				t.Error(err)
				break
			}
		}

		//log.Print("copying ", v, "...")

		r := src.Reader(v, nil, nil)
		w := dst.Writer(v, nil, nil)
		//log.Print("reader:", r)
		//log.Print("writer:", w)
		buf := r.Zero(-1)
		rc := 0
		for {
			nr, err := r.Read(buf)
			switch bb := buf.(type) {
			case []int8:
				buf = bb[:nr]
			case []int16:
				buf = bb[:nr]
			case []int32:
				buf = bb[:nr]
			case []float32:
				buf = bb[:nr]
			case []float64:
				buf = bb[:nr]
			default:
				t.Error("bad buffer type", buf)
			}
			nw, erw := w.Write(buf)

			//log.Printf("read: %v/%v write: %v/%v", nr, err, nw, erw)
			rc += nr

			if nr != nw || (erw != nil && erw != io.EOF) {
				t.Errorf("read: %v/%v write: %v/%v", nr, err, nw, erw)
				break
			}
			if err == io.EOF {
				break
			}
		}

		exp := 1
		for _, v := range dst.Header.vars[i].lengths {
			if v == 0 {
				exp *= int(numrecs)
			} else {
				exp *= v
			}
		}
		if rc != exp {
			t.Error("copied ", rc, " values, expected ", exp)
		}
	}

	err = UpdateNumRecs(dstf)
	if err != nil {
		t.Error("updating numrecs", err)
	}

	// compare

	srcf.Seek(0, 0)
	dstf.Seek(0, 0)

	srcd, err := ioutil.ReadAll(srcf)
	dstd, err := ioutil.ReadAll(dstf)

	if len(srcd) != len(dstd) {
		t.Error(srcpath, ":different lengths", len(srcd), len(dstd))
	}

	d := 0
	for i := 0; i < len(srcd) && i < len(dstd); i++ {
		if srcd[i] != dstd[i] {
			t.Error(srcpath, ":difference at offset ", i)
			d++
		}
		if d > 10 {
			t.Error(srcpath, ": too many differences")
			break
		}
	}
}
