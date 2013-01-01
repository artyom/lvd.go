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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func testAllFiles(t *testing.T, tf func(string, *testing.T)) {
	dir := os.Getenv("NETCDF_TESTDIR")
	if dir == "" {
		dir = "./testdata"
	}

	pattern := filepath.Join(dir, "*.nc")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}

	if files == nil {
		t.Fatal("No match for pattern " + filepath.Join(dir, "*.nc"))
	}

	for _, f := range files {
		log.Print("Testing on input ", f, "...")
		tf(f, t)
	}
}

func TestBinaryCompatibility(t *testing.T) {
	testAllFiles(t, readWriteCompareHeader)
}

func readWriteCompareHeader(srcpath string, t *testing.T) {
	srcf, err := os.Open(srcpath)
	if err != nil {
		t.Error(err)
		return
	}

	src, err := ReadHeader(srcf)
	if err != nil {
		t.Error(err)
		return
	}

	if errs := src.Check(); errs != nil {
		fmt.Println(src)
		t.Errorf("%v", errs)
		return
	}

	dst := NewHeader(src.Dimensions(""), src.Lengths(""))

	for _, a := range src.Attributes("") {
		dst.AddAttribute("", a, src.GetAttribute("", a))
	}

	for _, v := range src.Variables() {
		dst.AddVariable(v, src.Dimensions(v), src.ZeroValue(v, 0))
		for _, a := range src.Attributes(v) {
			dst.AddAttribute(v, a, src.GetAttribute(v, a))
		}
	}

	// cheat; normally this is done by Define, but we need bit for bit equality
	dst.fixRecordStrides()
	dst.version = src.version
	dst.setOffsets(src.dataStart())

	fi, err := srcf.Stat()
	if err != nil {
		t.Error(err)
		return
	}
	dst.setNumRecs(fi.Size())
	if dst.numrecs != src.numrecs {
		t.Errorf("computed numrecs %d != original numrecs %d", dst.numrecs, src.numrecs)
		dst.numrecs = src.numrecs
	}

	dstf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Error(err)
		return
	}

	//	log.Println("tmp file: ", dstf.Name())
	defer os.Remove(dstf.Name())

	if err := dst.WriteHeader(dstf); err != nil {
		t.Error(err)
		return
	}

	if errs := dst.Check(); errs != nil {
		fmt.Println(dst)
		t.Errorf("%v", errs)
		return
	}

	// compare

	srcf.Seek(0, 0)
	dstf.Seek(0, 0)

	srcd, err := ioutil.ReadAll(srcf)
	dstd, err := ioutil.ReadAll(dstf)

	for i := 0; i < len(srcd) && i < len(dstd); i++ {
		if srcd[i] != dstd[i] {
			t.Error(srcpath, ":difference at offset ", i)
			break
		}
	}
}
