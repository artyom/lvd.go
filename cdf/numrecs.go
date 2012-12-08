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

// this file contains the code to deal with the numrecs field in cdf headers.

package cdf

import (
	"encoding/binary"
	"os"
)

const _NumRecsOffset = 4 // position of the bigendian int32 in the header

// UpdateNumRecs determines the number of record from the file size and
// writes it into the file's header as the 'numrecs' field.
//
// Any incomplete trailing record will not be included in the count.
//
// Only valid headers will be updated.
// After a succesful call f's filepointer will be left at the end of the file.
//
// This library does not use the numrecs header field but updating it
// enables full bit for bit compatibility with other libraries.  There
// is no need to call this function until after all updates by the program,
// and it is rather costly because it reads, parses and checks the entire header.
func UpdateNumRecs(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if _, err = f.Seek(0, 0); err != nil {
		return err
	}

	h, err := ReadHeader(f)
	if err != nil {
		return err
	}

	if errs := h.Check(); errs != nil {
		return errs[0] // only room for the first
	}

	h.setNumRecs(fi.Size())

	if _, err = f.Seek(_NumRecsOffset, 0); err != nil {
		return err
	}

	if err = binary.Write(f, binary.BigEndian, h.numrecs); err != nil {
		return err
	}

	if _, err = f.Seek(0, 2); err != nil {
		return err
	}

	return nil
}

// setNumRecs computes the number or records from the filesize and sets the 
// header field accordingly.  Returns the real number of records.
// For fsize < 0, sets numrecs to -1 and returns -1.
func (h *Header) setNumRecs(fsize int64) int64 {
	if fsize < 0 {
		h.numrecs = -1
		return -1
	}

	offs, size := h.slabs()

	if size == 0 || fsize < offs {
		h.numrecs = 0
		return 0
	}

	nr := (fsize - offs) / size

	if nr < (1 << 31) {
		h.numrecs = int32(nr)
	} else {
		h.numrecs = -1
	}

	return nr
}
