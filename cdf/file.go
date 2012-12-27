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

// This file contains the File type.

package cdf

import (
	"bytes"
	"io"
)

// A ReaderWriterAt is the underlying storage for a NetCDF file,
// providing {Read,Write}At([]byte, int64) methods.
// Since {Read,Write}At are required to not modify the underlying
// file pointer, one instance may be shared by multiple Files, although
// the documentation of io.WriterAt specifies that it only has to 
// guarantee non-concurrent calls succeed.
type ReaderWriterAt interface {
	io.ReaderAt
	io.WriterAt
}

type File struct {
	rw     ReaderWriterAt
	Header *Header
}

// Open reads the header from an existing storage rw and returns a File
// usable for reading or writing (if the underlying rw permits).
func Open(rw ReaderWriterAt) (*File, error) {
	h, err := ReadHeader(io.NewSectionReader(rw, 0, 1<<31))
	if err != nil {
		return nil, err
	}
	return &File{rw: rw, Header: h}, nil
}

// Create writes the header to a storage rw and returns a File
// usable for reading and writing.
//
// The header should not be mutable, and may be shared by multiple
// Files(*).  Note in this case that at every Create the headers numrec
// field will be reset to -1 (STREAMING).
func Create(rw ReaderWriterAt, h *Header) (*File, error) {
	if h.isMutable() {
		panic("Create must be called with a fully defined header")
	}
	nr := h.numrecs
	h.numrecs = _STREAMING // (*) potential race
	var buf bytes.Buffer
	err := h.WriteHeader(&buf)
	h.numrecs = nr
	if err != nil {
		return nil, err
	}
	if _, err := rw.WriteAt(buf.Bytes(), 0); err != nil {
		return nil, err
	}
	return &File{rw: rw, Header: h}, nil
}
