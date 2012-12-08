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

// This file contains the code to write CDF variable data

package cdf

import ()

// A writer is an object that can write values to a CDF file.
type Writer interface {
	// Write writes len(values.([]T)) elements from values to the underlying file.
	//
	// Values must be a slice of int{8,16,32} or float{32,64} or a
	// string, according to the type of the variable.  if n <
	// len(values.([]T)), err will be set.
	Write(values interface{}) (n int, err error)
}

// Create a writer
func (f *File) Writer(v string, begin, end []int) Writer {
	return nil
}
