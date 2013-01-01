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

package cdf

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestReader(t *testing.T) {
	h := NewHeader([]string{"time", "Z", "Y", "X"}, []int{0, 5, 7, 9})
	h.AddAttribute("", "info", "This is a testfile")
	h.AddVariable("z", []string{"Z"}, []int8{})
	h.AddVariable("y", []string{"Y"}, []int16{})
	h.AddVariable("x", []string{"X"}, []int32{})
	h.AddVariable("f", []string{"time", "X", "Y", "Z"}, []float32{})
	h.AddVariable("g", []string{"X", "Y", "Z"}, []int32{})
	h.Define()

	//log.Print(h)

	dstf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}

	//log.Println("tmp file: ", dstf.Name())
	defer os.Remove(dstf.Name())

	dst, err := Create(dstf, h)
	if err != nil {
		t.Fatal(err)
	}

	n, err := dstf.Seek(0, 2)
	//log.Println("header ends: ", n)
	if err != nil {
		t.Fatal(err)
	}

	// pad to datastart
	for nn := h.dataStart(); n < nn; n++ {
		err := binary.Write(dstf, binary.BigEndian, uint8(0x80))
		if err != nil {
			t.Fatal(err)
		}
	}

	// write z: 5 int8's
	for i := 0; i < 5; i++ {
		err := binary.Write(dstf, binary.BigEndian, int8(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	// pad
	for i := 0; i < 3; i++ {
		err := binary.Write(dstf, binary.BigEndian, int8(-1))
		if err != nil {
			t.Fatal(err)
		}
	}

	// write y: 7 int16's
	for i := 0; i < 7; i++ {
		err := binary.Write(dstf, binary.BigEndian, int16(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	// pad
	for i := 0; i < 1; i++ {
		err := binary.Write(dstf, binary.BigEndian, int16(-1))
		if err != nil {
			t.Fatal(err)
		}
	}

	// write x: 9 int32's
	for i := 0; i < 9; i++ {
		err := binary.Write(dstf, binary.BigEndian, int32(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	// write g, the non-record variable
	for x := 0; x < 9; x++ {
		for y := 0; y < 7; y++ {
			for z := 0; z < 5; z++ {
				err := binary.Write(dstf, binary.BigEndian, int32(z*100+y*10+x))
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	// write f, the record variable
	for tt := 0; tt < 5; tt++ {
		for x := 0; x < 9; x++ {
			for y := 0; y < 7; y++ {
				for z := 0; z < 5; z++ {
					err := binary.Write(dstf, binary.BigEndian, float32(tt*1000+z+y*10+x*100))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}

	fi, err := dstf.Stat()
	if err != nil {
		t.Fatal(err)
	}
	nr := h.setNumRecs(fi.Size())
	if nr != 5 {
		t.Error("filesize: ", fi.Size(), " numrecs: ", nr, " != 5")
	}

	{
		r := dst.Reader("x", nil, nil)
		buf := r.Zero(-1)
		if b, ok := buf.([]int32); !ok || len(b) != 9 {
			t.Fatal("bad Zero for x", len(b))
		}
		n, err := r.Read(buf)
		if n != 9 || err != nil {
			t.Error("reading x: ", n, err)
		}
		for i, v := range buf.([]int32) {
			if int(v) != i {
				t.Error("bad x: ", buf)
				break
			}
		}
		n, err = r.Read(buf)
		if err != io.EOF {
			t.Error("read x past eof:", buf)
		}

	}

	{
		r := dst.Reader("y", nil, nil)
		buf := r.Zero(-1)
		if b, ok := buf.([]int16); !ok || len(b) != 7 {
			t.Fatal("bad Zero for y")
		}
		n, err := r.Read(buf)
		if n != 7 || err != nil {
			t.Error("reading y: ", n, err)
		}
		for i, v := range buf.([]int16) {
			if int(v) != i {
				t.Error("bad y: ", buf)
				break
			}
		}
		n, err = r.Read(buf)
		if err != io.EOF {
			t.Error("read y past eof:", buf)
		}

	}

	{
		r := dst.Reader("z", nil, nil)
		buf := r.Zero(-1)
		if b, ok := buf.([]int8); !ok || len(b) != 5 {
			t.Fatal("bad Zero for z")
		}
		n, err := r.Read(buf)
		if n != 5 || err != nil {
			t.Error("reading y: ", n, err)
		}
		for i, v := range buf.([]int8) {
			if int(v) != i {
				t.Error("bad z: ", buf)
				break
			}
		}
		n, err = r.Read(buf)
		if err != io.EOF {
			t.Error("read z past eof:", buf)
		}
	}

	{
		r := dst.Reader("f", nil, nil)
		buf := r.Zero(-1)
		if b, ok := buf.([]float32); !ok || len(b) != 5*7*9 {
			t.Fatal("bad Zero for f")
		}

		for tt := 0; tt < 5; tt++ {

			n, err := r.Read(buf)
			if n != 5*7*9 || err != nil {
				t.Error("reading f: ", n, err)
			}

			ch := make(chan float32)
			go func() {
				for _, v := range buf.([]float32) {
					ch <- v
				}
				close(ch)
			}()

		cmp:
			for x := 0; x < 9; x++ {
				for y := 0; y < 7; y++ {
					for z := 0; z < 5; z++ {
						v := <-ch
						if v != float32(1000*tt+z+y*10+x*100) {
							log.Print(" f[:", len(buf.([]float32)), "]: ", buf)
							break cmp
						}
					}
				}
			}
		}

		n, err := r.Read(buf)
		if n != 0 || err != io.EOF {
			t.Error("last reading f: ", n, err)
		}

	}

}
