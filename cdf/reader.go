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

// This file contains the methods to read CDF variable data.

package cdf

import (
	"encoding/binary"
	"errors"
	"io"
)

// A reader is an object that can read values from a CDF file.
type Reader interface {
	// Read reads len(values.([]T)) elements from the underlying file into values.
	//
	// Values must be a slice of int{8,16,32} or float{32,64},
	// corresponding to the type of the variable, with one
	// exception: A variable of NetCDF type CHAR must be read into
	// a []byte.  Read returns the number of elements actually
	// read.  if n < len(values.([]T)), err will be set.
	Read(values interface{}) (n int, err error)

	// Zero returns a slice of the appropriate type for Read
	// if n < 0, the slice will be of the length
	// that can be read contiguously.
	Zero(n int) interface{}
}

// Create a reader that starts at the corner begin, ends at end and 
// steps through the matrix with the given strides.  If begin is nil,
// it defaults to the origin (0, 0, ...).  If end is nil, it defaults
// to the f.Header.Lengths(v).
func (f *File) Reader(v string, begin, end []int) Reader {
	vv := f.Header.varByName(v)
	if vv == nil {
		return nil
	}

	if begin != nil && len(begin) != len(vv.dim) {
		panic("invalid begin index vector")
	}

	if end != nil && len(end) != len(vv.dim) {
		panic("invalid end index vector")
	}

	var b, e, sz, sk int64

	if begin != nil {
		b = vv.offsetOf(begin)
	} else {
		b = vv.begin
	}

	if end != nil {
		e = vv.offsetOf(end)
	} else if !vv.isRecordVariable() {
		e = vv.offsetOf(vv.lengths)
	}

	if !vv.isRecordVariable() {
		sz = e - b
		sk = e - b
	} else {
		sz = vv.strides[0] // vsize
		sk = vv.strides[1] // slabsize
	}

	switch vv.dtype {
	case _BYTE, _CHAR:
		return &int8Reader{f.rw, b, e, sz, sk, b}
	case _SHORT:
		return &int16Reader{f.rw, b, e, sz, sk, b}
	case _INT:
		return &int32Reader{f.rw, b, e, sz, sk, b}
	case _FLOAT:
		return &float32Reader{f.rw, b, e, sz, sk, b}
	case _DOUBLE:
		return &float64Reader{f.rw, b, e, sz, sk, b}
	}
	panic("invalid variable data type")
}

type stridedReader struct {
	r                  io.ReaderAt
	begin, end         int64
	stripesize, stride int64
	curr               int64
}

func (r *stridedReader) relOffs(elemsz int) int64 {
	s := (r.curr - r.begin) / r.stride // stripe number
	e := r.curr - r.begin - s*r.stride // offset within stripe
	nn := (s * r.stripesize) + e
	nn /= int64(elemsz)
	return nn
}

func (r *stridedReader) Read(p []byte) (n int, err error) {
	se := (r.curr - r.begin) / r.stride // stripe number
	se = r.begin + se*r.stride          // stripe begin
	se += r.stripesize                  // stripe end

	for len(p) > 0 {
		nn := int64(len(p))
		if r.curr+nn > se {
			nn = se - r.curr
		}
		if r.end > 0 && r.curr+nn > r.end {
			nn = r.end - r.curr
		}

		nr, err := r.r.ReadAt(p[:nn], r.curr)
		r.curr += int64(nr)
		n += nr
		p = p[nr:]
		if r.curr == se {
			r.curr += r.stride - r.stripesize
			se += r.stride
		}
		if err != nil {
			return n, err
		}
		if r.curr == r.end {
			return n, io.EOF
		}
	}

	return n, nil
}

func (r *stridedReader) readElems(elemsz int, values interface{}) (int, error) {
	nn := r.relOffs(elemsz)
	err := binary.Read(r, binary.BigEndian, values)
	return int(r.relOffs(elemsz) - nn), err
}

var badValueType = errors.New("value type mismatch")

type int8Reader stridedReader
type int16Reader stridedReader
type int32Reader stridedReader
type float32Reader stridedReader
type float64Reader stridedReader

func (r *int8Reader) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]int8); !ok {
		return 0, badValueType
	}
	return (*stridedReader)(r).readElems(1, values)
}

func (r *int16Reader) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]int16); !ok {
		return 0, badValueType
	}
	return (*stridedReader)(r).readElems(2, values)
}

func (r *int32Reader) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]int32); !ok {
		return 0, badValueType
	}
	return (*stridedReader)(r).readElems(4, values)
}

func (r *float32Reader) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]float32); !ok {
		return 0, badValueType
	}
	return (*stridedReader)(r).readElems(4, values)
}

func (r *float64Reader) Read(values interface{}) (n int, err error) {
	if _, ok := values.([]float64); !ok {
		return 0, badValueType
	}
	return (*stridedReader)(r).readElems(8, values)
}

func (r *int8Reader) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize)
	}
	return make([]int8, n)
}

func (r *int16Reader) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 2)
	}
	return make([]int16, n)
}

func (r *int32Reader) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 4)
	}
	return make([]int32, n)
}

func (r *float32Reader) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 4)
	}
	return make([]float32, n)
}
func (r *float64Reader) Zero(n int) interface{} {
	if n < 0 {
		n = int(r.stripesize / 8)
	}
	return make([]float64, n)
}
