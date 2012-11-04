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
	"math/rand"
	"testing"
)

func TestCellLevel(t *testing.T) {
	if v := cell63(1).level(); v != 0 {
		t.Error("Expecting (1).level() == 0, got ", v)
	}
	if v := cell63(2).level(); v != 1 {
		t.Error("Expecting (2).level() == 1, got ", v)
	}
	if v := cell63(3).level(); v != 0 {
		t.Error("Expecting (3).level() == 0, got ", v)
	}
	if v := cell63(4).level(); v != 2 {
		t.Error("Expecting (4).level() == 2, got ", v)
	}
	if v := cell63(8).level(); v != 3 {
		t.Error("Expecting (8).level() == 3, got ", v)
	}
	if v := cell63(0x10).level(); v != 4 {
		t.Error("Expecting (0x10).level() == 4, got ", v)
	}
}

func TestCell(t *testing.T) {
	for _, tc := range []struct {
		c, lsb                       cell63
		level                        int
		parent, left, right, sibling cell63
		begin, end                   uint64
	}{
		{0x0, 0x0, 0, 0x0, 0x0, 0x0, 0x0, 0, 0},
		{0x1, 0x1, 0, 0x2, 0x1, 0x1, 0x3, 0, 1}, // children = self
		{0x2, 0x2, 1, 0x4, 0x1, 0x3, 0x6, 0, 2},
		{0x4, 0x4, 2, 0x8, 0x2, 0x6, 0xc, 0, 4},
		{0x5, 0x1, 0, 0x6, 0x5, 0x5, 0x7, 2, 3},                              // children = self
		{unity63, unity63, 63, 0, 0x4 << 60, 0xc << 60, unity63, 0, 1 << 63}, // no parent, it's own sibling
	} {

		if v := tc.c.lsb(); v != tc.lsb {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").lsb() == ", tc.lsb, ", got ", v)
		}
		if v := tc.c.level(); v != tc.level {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").level() == ", tc.level, ", got ", v)
		}
		if v := tc.c.parent(); v != tc.parent {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").parent() == ", tc.parent, ", got ", v)
		}
		if v, w := tc.c.children(); v != tc.left || w != tc.right {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").children() == ", tc.left, "and", tc.right, ", got ", v, w)
		}
		if v := tc.c.sibling(); v != tc.sibling {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").sibling() == ", tc.sibling, ", got ", v)
		}
		if v := tc.c.begin(); v != tc.begin {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").begin() == ", tc.begin, ", got ", v)
		}
		if v := tc.c.end(); v != tc.end {
			t.Error(uint64(tc.c), "Expecting(", tc.c, ").end() == ", tc.end, ", got ", v)
		}

	}
}

func TestCellParent(t *testing.T) {
	for _, c := range []cell63{0, 1, 2, 34, 42, 99, 127, 132, 15000} {
		if !c.parent().contains(c) {
			t.Error("Expecting (", c, ").parent() ", c.parent(), " .contains(c)")
		}

		if !c.parent().contains(c.sibling()) {
			t.Error("Expecting (", c, ").parent() ", c.parent(), " .contains(c.sibling() ", c.sibling(), " )")
		}

		if c.sibling().parent() != c.parent() {
			t.Error("Expecting (", c, ").sibling().parent() ", c.sibling().parent(), " == c.parent() ", c.parent())
		}
	}
}

func TestCellSingleton(t *testing.T) {
	for _, e := range []int64{0, 1, 2, 34, 42, 99, 127, 132, 15000} {
		c := singleton(e)
		if v := c.lsb(); v != 1 {
			t.Error("Expecting c.lsb() == 1, got ", v)
		}
		if v := c.level(); v != 0 {
			t.Error("Expecting c.level() == 0, got ", v)
		}
		if v := c.begin(); v != uint64(e) {
			t.Error("Expecting c.begin() == ", e, ", got ", v)
		}
		if v := c.end(); v != uint64(e+1) {
			t.Error("Expecting c.end() == ", e+1, ", got ", v)
		}
		if !c.contains(c) {
			t.Error(c, " does not contain itself")
		}
	}
}

func TestSetEmpty(t *testing.T) {
	s := NewSet63()
	if !s.IsEmpty() {
		t.Error("Emtpy set not empty", s)
	}

	if v := s.Count(); v != 0 {
		t.Error("Emtpy set count", v, s)
	}

	s.ForEach(func(e int64) bool { 
		t.Error("Emtpy set has element", e)
		return true
	})

	s.ForEachInterval(func(b, e int64) bool {
		t.Error("Emtpy set has interval", b, e)
		return true
	})
	if !s.Union(s).IsEmpty() {
		t.Error("Emtpy set ∩ itself not empty", s)
	}
	if !s.Intersection(s).IsEmpty() {
		t.Error("Emtpy set ∪ itself not empty", s)
	}
}

func TestSetFull(t *testing.T) {
	s := Set63{unity63}
	if min, max := s.Span(); min != 0 || max != 1<<63-1 {
		t.Errorf("Expected full span = %s, got %x, %x", unity63, min, max)
	}
	if v := s.Count(); v != 1<<63 {
		t.Errorf("Expected full count = 1<<63, got %x", v)
	}
	if v := NewSet63().Complement(); !v.Equal(s) {
		t.Error("empty complement ", []cell63(v), "is not full", []cell63(s))
	}
	if v := s.Complement(); !v.IsEmpty() {
		t.Error("full complement ", []cell63(v), "is not empty")
	}
}

func TestSet1(t *testing.T) {
	s := NewSet63(1, 2, 3, 4, 3, 2, 1, 16, 18, 17, 19)
	if s.IsEmpty() {
		t.Error("Emtpy set", s)
	}

	if v := s.unnormalized(); len(v) > 0 {
		t.Fatal(s, " unnormalized: ", v)
	}

	if s.Equal(NewSet63()) || NewSet63().Equal(s) {
		t.Error(s, " equal to emtpy set")
	}

	if v := len(s); v != 4 {
		t.Error("len(s)", v, []cell63(s))
	}

	if v := s.Count(); v != 8 {
		t.Error("Set count", v, s)
	}

	if min, max := s.Span(); min != 1 || max != 19 {
		t.Error("Set span bad", min, max, s)
	}

	v := []int64{1, 2, 3, 4, 16, 17, 18, 19}
	for _, e := range v {
		if !s.Contains(e) {
			t.Error("Set fails to contain element", e)
		}
	}

	s.ForEach(func(e int64) bool {
		if e != v[0] {
			t.Error("Set has wrong element", e)
			return false
		}
		v = v[1:]
		return true
	})

	ch := make(chan struct { Begin, End int64 })
	go func() {
		s.ForEachInterval(func(b, e int64) bool {
			ch <- struct{Begin,End int64}{b, e}
			return true
		})
		close (ch)
	}()
	e, ok := <-ch
	if !ok {
		t.Error("set ", []cell63(s), " has no interval")
	}
	if e.Begin != 1 || e.End != 4 {
		t.Error("set ", []cell63(s), " has wrong interval", e)
	}

	e, ok = <-ch
	if !ok {
		t.Error("set ", []cell63(s), " has no second interval")
	}
	if e.Begin != 16 || e.End != 19 {
		t.Error("set ", []cell63(s), " has wrong interval", e)
	}
	e, ok = <-ch
	if ok {
		t.Error("set ", []cell63(s), " has third interval", e)
	}
}

func TestInterval(t *testing.T) {
	for n := 0; n < 1000; n++ {
		min, max := rand.Int63n(64), rand.Int63n(64)
		if min > max {
			min, max = max, min
		}
		s := Interval(min, max)
		if c := s.Count(); c != uint64(max-min+1) {
			t.Error(s, " bad count ", c, " != ", max-min+1)
		}
		for i := int64(0); i < 64; i++ {
			if (min <= i && i <= max) != s.Contains(i) {
				t.Error(s, " has wrong element", i)
			}
		}
		s.ForEachInterval(func(b, e int64) bool {
			if b != min || e != max {
				t.Error("set ", s, " has wrong interval", b, e)
			}
			return true
		})
	}
}

// generate a bitmask of random stretches of zeroes and ones
func genSet6(down, up int) (r uint64) {
	for m := uint64(1); m != 0; m <<= uint(rand.Intn(down)) {
		u := rand.Intn(up)
		for i := 0; i < u && m != 0; i++ {
			r |= m
			m <<= 1
		}
	}
	return
}

// turn a 64 bit bitmask into a set
func set6toSet63(x uint64) Set63 {
	var e []int64
	i := int64(0)
	for ; x != 0; x >>= 1 {
		if x&1 != 0 {
			e = append(e, i)
		}
		i++
	}
	return NewSet63(e...)
}

func TestUnion(t *testing.T) {
	s1 := NewSet63(0, 1, 5)
	s2 := NewSet63(0, 1, 2, 3, 4)
	su := NewSet63(0, 1, 2, 3, 4, 5)
	u := s1.Union(s2)
	if un := u.unnormalized(); len(un) > 0 {
		t.Fatal(u, "unnormalized: ", un)
	} else if !u.Equal(su) {
		t.Error(s1, " ∪ ", s2, " == ", u, " expecting: ", su)
	}
	if u = s1.Union(s1); !u.Equal(s1) {
		t.Error(s1, " ∪ itself == ", u, " != ", s1)
	}
	if u = s2.Union(s2); !u.Equal(s2) {
		t.Error(s2, " ∪ itself == ", u, " != ", s2)
	}
}

func TestUnion2(t *testing.T) {
	s1 := set6toSet63(7526070234921300089)
	s2 := set6toSet63(14055315658856709368)
	su := set6toSet63(7526070234921300089 | 14055315658856709368)
	u := s1.Union(s2)
	if un := u.unnormalized(); len(un) > 0 {
		t.Fatal(u, "unnormalized: ", un)
	} else if !u.Equal(su) {
		t.Error(s1, " ∪ ", s2, " == ", u, " expecting: ", su)
	}
}

func TestUnionRandom(t *testing.T) {
	for n := 0; n < 1000; n++ {
		x1, x2 := genSet6(4, 4), genSet6(6, 6)
		s1, s2 := set6toSet63(x1), set6toSet63(x2)
		xu := x1 | x2
		su := set6toSet63(xu)
		u := s1.Union(s2)
		if un := u.unnormalized(); len(un) > 0 {
			t.Fatal(u, "unnormalized: ", un)
		}
		u.ForEach(func(e int64) bool {
			xu ^= (1 << (uint64(e)))
			return true
		})
		if xu != 0 {
			t.Fatal(x1, x2, ": ", s1, "   ∪   ", s2, " == ", u, " expecting: ", su)
		}
	}
}

func TestIntersection(t *testing.T) {
	s1 := NewSet63(0, 1, 5)
	s2 := NewSet63(0, 1, 2, 3, 4)
	su := NewSet63(0, 1)
	u := s1.Intersection(s2)
	if un := u.unnormalized(); len(un) > 0 {
		t.Fatal(u, "unnormalized: ", un)
	}
	if !u.Equal(su) {
		t.Error(s1, " ∩ ", s2, " == ", u, " expecting: ", su)
	}
	if !s1.Intersects(s2) && !su.IsEmpty() {
		t.Error(s1, " ∩  ", s2, "  == ∅ but ", su, "is not empty")
	}
	if u = s1.Intersection(s1); !u.Equal(s1) {
		t.Error(s1, " ∩ itself == ", u, " != ", s1)
	}
	if u = s2.Intersection(s2); !u.Equal(s2) {
		t.Error(s2, " ∩ itself == ", u, " != ", s2)
	}

}

func TestIntersection2(t *testing.T) {
	s1 := set6toSet63(2152115872029880693 & 0xffff)
	s2 := set6toSet63(8943652414326511639 & 0xffff)
	su := set6toSet63(2152115872029880693 & 8943652414326511639 & 0xffff)
	u := s1.Intersection(s2)
	if un := u.unnormalized(); len(un) > 0 {
		t.Fatal(u, "unnormalized: ", un)
	}
	if !u.Equal(su) {
		t.Error(s1, "   ∩   ", s2, " == ", u, " expecting: ", su)
	}
	if !s1.Intersects(s2) && !su.IsEmpty() {
		t.Error(s1, " ∩  ", s2, "  == ∅ but ", su, "is not empty")
	}

}

func TestIntersectionRandom(t *testing.T) {
	for n := 0; n < 1000; n++ {
		x1, x2 := genSet6(4, 4), genSet6(6, 6)
		s1, s2 := set6toSet63(x1), set6toSet63(x2)
		xu := x1 & x2
		su := set6toSet63(xu)
		u := s1.Intersection(s2)
		if un := u.unnormalized(); len(un) > 0 {
			t.Fatal(u, "unnormalized: ", un)
		}
		if !s1.Intersects(s2) && !su.IsEmpty() {
			t.Error(s1, " ∩  ", s2, "  == ∅ but ", su, "is not empty")
		}
		u.ForEach(func(e int64) bool {
			xu ^= (1 << (uint64(e)))
			return true
		})
		if xu != 0 {
			t.Fatal(x1, x2, ": ", s1, "   ∩   ", s2, " == ", u, " expecting: ", su)
		}
	}
}

func TestComplement(t *testing.T) {
	s1 := NewSet63(0, 1, 5)
	s2 := Interval(6, 1<<63-1)
	su := NewSet63(2, 3, 4).Union(s2)
	u := s1.Complement()
	if un := u.unnormalized(); len(un) > 0 {
		t.Fatal(u, "unnormalized: ", un)
	}
	if !u.Equal(su) {
		t.Error(s1, " complement  == ", u, " expecting: ", su)
	}
}

func TestComplementRandom(t *testing.T) {
	for n := 0; n < 1000; n++ {
		x1 := genSet6(4, 4)
		s1 := set6toSet63(x1).Union(Interval(64, 1<<63-1))
		xu := ^x1
		su := set6toSet63(xu)
		u := s1.Complement()
		if un := u.unnormalized(); len(un) > 0 {
			t.Fatal(u, "unnormalized: ", un)
		}
		if v := uint64(u.Count()) + uint64(s1.Count()); v != 1<<63 {
			t.Error("Don't add up: ", u.Count(), s1.Count(), 1<<63-v)
		}
		u.ForEach(func(e int64) bool {
			if e > 63 {
				t.Fatal("element larger than 63:", e)
			}
			xu ^= (1 << (uint64(e)))
			return true
		})
		if xu != 0 {
			t.Fatal(x1, ": ", s1, " complement == ", u, " expecting: ", su)
		}
	}
}
