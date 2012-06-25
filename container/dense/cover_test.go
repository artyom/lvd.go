// Copyright 2012 Google Inc. All Rights Reserved.
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

package dense

import (
	"testing"
//	"log"
)

func TestCover1(t *testing.T) {
	x := uint64(252201579162140679)
	s := set6toSet63(x)
	minGrain := uint64(1)
	maxSize := 5

	c := s.Cover(maxSize, minGrain)

	if v := c.unnormalized(); len(v) > 0 {
		t.Fatal(x, s, " cover(", maxSize, minGrain, "): ", []cell63(c), " unnormalized: ", v)
	}
	if v := c.Intersection(s); !v.Equal(s) {
		t.Error(v, " bad cover: intersection, expected ", s)
	}
	if v := c.Union(s); !v.Equal(c) {
		t.Error(v, " bad cover: union expected ", c)
	}
	if 0 < maxSize && maxSize < len(c) {
		t.Error([]cell63(c), " contains more than ", maxSize, " elements")
	}
	for _, e := range(c) {
		if uint64(e.lsb()) < minGrain {
			t.Error([]cell63(c), " contains cell smaller than ", minGrain)
		}
	}
}

func TestCover(t *testing.T) {
//	s := NewSet63(1,2,3,5,6,7,15, 20,21,22,23, 33,34,35,36)
	for i := 0; i < 1000; i++ {
		x := genSet6(8,4)
		s := set6toSet63(x)
		for minGrain := uint64(1); minGrain < 1025; minGrain <<= 1 {
			for maxSize := 1; maxSize < 12; maxSize++ {
				c := s.Cover(maxSize, minGrain)
				if v := c.unnormalized(); len(v) > 0 {
					t.Fatal(x, s, " cover(", maxSize, minGrain, "): ", []cell63(c), " unnormalized: ", v)
				}
				if v := c.Intersection(s); !v.Equal(s) {
					t.Error(v, " bad cover: intersection, expected ", s)
				}
				if v := c.Union(s); !v.Equal(c) {
					t.Error(v, " bad cover: union expected ", c)
				}
				if 0 < maxSize && maxSize < len(c) {
					t.Error([]cell63(c), " contains more than ", maxSize, " elements")
				}
				for _, e := range(c) {
					if uint64(e.lsb()) < minGrain {
						t.Error([]cell63(c), " contains cell smaller than ", minGrain)
					}
				}
			}
		}
//		log.Println("ok: ", s)
	}
}
