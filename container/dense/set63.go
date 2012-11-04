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

// Package dense provides types to represent dense sets of integers.
package dense

import (
	"bytes"
	"fmt"
	//"log"
	"sort"
)

// Internal computations are on uint64 and halfopen sets.
// public elements and intervals are int64 and closed sets.

// A cell63 represents a contiguous halfopen interval [b, e)
// where b and e share a common binary prefix and vary in the suffix.
// the lenght of the suffix is the level. The cell id is the prefix
// suffixed by 10000..., i.e. the position of the least significant bit
// determines the level.  Since the last bit is always 1, a set of level 0,
// has the single element n and is presented as n<<1 + 1.
type cell63 uint64

const unity63 cell63 = 1 << 63 // the whole interval

// The lsb is the trailing 1 that signifies the level.
// as a number, it is equal to the size of the cell.
func (c cell63) lsb() cell63 { return c & -c }

// The level of the set is the position of the lsb.
func (c cell63) level() (l int) {
	c = c & -c                     // lsb
	if c&0xaaaaaaaaaaaaaaaa != 0 { // 101010..
		l += 1
	}
	if c&0xcccccccccccccccc != 0 { // 11001100....
		l += 2
	}
	if c&0xf0f0f0f0f0f0f0f0 != 0 { // 11110000...
		l += 4
	}
	if c&0xff00ff00ff00ff00 != 0 {
		l += 8
	}
	if c&0xffff0000ffff0000 != 0 {
		l += 16
	}
	if c&0xffffffff00000000 != 0 {
		l += 32
	}
	return
}

// The cell c contains d if they have the same prefix upto
// but not including c's lsb.  for all c, c.contains(c)
func (c cell63) contains(d cell63) bool {
	return (c^d) & ^((c&-c)<<1-1) == 0
}

// The sibling of a cell is the other half of the same parent,
// which can be obtained by flipping the lsb<<1.
// the unit63 cell is its own sibling.
func (c cell63) sibling() cell63 {
	return c ^ ((c & -c) << 1)
}

// The parent of a cell is the one with lsb<<1 set and trailing zeroes.
// The unity63 cell has zero parent.
func (c cell63) parent() (p cell63) {
	p = (c & -c) // avoid declaring a variable to permit inlining
	return (c ^ p) | p<<1
}

// The two children of c.  At level 1, both children are == c.
func (c cell63) children() (p, q cell63) {
	p = (c & -c) >> 1              // new trailing bit, zero for leaves
	c |= p                         // set trailing bit
	return c &^ (p << 1), c | p<<1 // clear and set former lsb, but not if c was leaf
}

// singleton constructs the cell63 containing only the single element e.
func singleton(e int64) cell63 { return cell63(e<<1) | 1 }

// begin is the first element of the cell.
func (c cell63) begin() uint64 { return uint64((c - (c & -c)) >> 1) }

// end is one past the last element of the cell c+(c&-c)) >> 1 would overflow for 0x80...
func (c cell63) end() uint64 { return uint64((c-(c&-c))>>1 + (c & -c)) }

// A Set63 represents dense sets of integers [0...2^63-1].
//
// The representation is efficient for 'lumpy' sets, sets
// where the probability for a number to be in the set is positively
// correlated with that of it's neighbours.  The Set63 will
// store consecutive runs of elements in 'cells' of size a power of two.
// For non-lumpy sets, a Set63 behaves like a sorted slice of elements.
//
// Many operations on a Set63 take time or space depending its
// the storage size, which is accessible as len(s).
//
// The price to pay for the efficient implementation is that inserting
// or removing a single element is rather costly.  The only way to do
// that is to take the union with a singleton, or the intersection
// with a singleton's complement.
type Set63 []cell63

// Construct a new Set63 out of the elements provided.
// All elements should be non-negative, a violation will cause a panic.
func NewSet63(elem ...int64) Set63 {
	if len(elem) == 0 {
		return nil
	}
	sort.Sort(int64Slice(elem))
	if elem[0] < 0 {
		panic("Set63 can not contain negative elements")
	}
	ss := make(Set63, 0, len(elem))
	for _, ee := range elem {
		e := singleton(ee)
		l := len(ss) - 1

		if l >= 0 && ss[l].contains(e) {  // includes the case ss[l] == e
			continue
		}

		for l >= 0 && e.contains(ss[l]) {
			ss, l = ss[:l], l-1
		}

		//  l >= 0 && ss[l] == e.sibling() {
		for l >= 0 && ss[l]^e^e.lsb()<<1 == 0 {
			ss, l = ss[:l], l-1
			e = e.parent()
		}
		ss = append(ss, e)
	}
	return ss
}

/*
func (s Set63) head()  (sh, st Set63) {
	if len(s) == 0 {
		return nil, s
	}
	var i int
	for i = 1; i < len(s); i++ {
		if s[i-1].end() != s[i].begin() {
			break
		}
	}
	return s[:i], s[i:]
}
*/

// return s' first contiguous interval and the remainder, and the first begin and end
// NOTE if we relax this to < and keep max end we could do operations on unnormalized (but still sorted) sets almost as efficiently 
func (s Set63) headx() (sh, st Set63, b, e uint64) {
	if len(s) == 0 {
		return nil, s, 0, 0
	}
	var i int
	for i = 1; i < len(s); i++ {
		if s[i-1].end() != s[i].begin() {
			break
		}
	}
	b, e = s[0].begin(), s[i-1].end()
	return s[:i], s[i:], b, e
}

// Union returns a Set63 containing all elements that are either in s or in t or both.
func (s Set63) Union(t Set63) Set63 {
	if t.IsEmpty() {
		return s
	}
	if s.IsEmpty() {
		return t
	}
	r := make(Set63, 0, len(s)+len(t)) // reasonable overestimate
	//log.Println()
	//log.Println([]cell63(s), " union ", []cell63(t))
	ss, s, sb, se := s.headx()
	tt, t, tb, te := t.headx()

	for sb != se && tb != te {

		//log.Println("ss: ", []cell63(ss), "  s: ", []cell63(s))
		//log.Println("tt: ", []cell63(tt), "  t: ", []cell63(t)) 
		//log.Println("from s: [", sb, ", ", se, ") from t: [", tb, ", ", te, ")")

		// 6 possibilities:
		// s is entirely contained in t
		if tb <= sb && se <= te {
			//log.Println("s entirely in t")
			ss, s, sb, se = s.headx()
			continue
		}

		// t is entirely contained in s
		if sb <= tb && te <= se {
			//log.Println("t entirely in s")
			tt, t, tb, te = t.headx()
			continue
		}

		// s comes entirely before t, and doesn't touch
		if se < tb {
			if ss != nil { // [sb, se) == ss
				//log.Println("s entirely before t, appending ", []cell63(ss))
				r = append(r, ss...)
			} else { // make new cover for [sb, se)
				//log.Println("s entirely before t, appending new cover [", sb, ", ", se, ")")
				r = append(r, interval(sb, se)...)
			}
			ss, s, sb, se = s.headx()
			continue
		}

		// t comes entirely before s, and doesn't touch
		if te < sb {
			if tt != nil { // [tb, te) == tt
				//log.Println("t entirely before s, appending ", []cell63(tt))
				r = append(r, tt...)
			} else { // make new cover for [tb, te)
				//log.Println("t entirely before s, appending new cover [", tb, ", ", te, ")")
				r = append(r, interval(tb, te)...)
			}
			tt, t, tb, te = t.headx()
			continue
		}

		// [sb ...[tb .. se)...te), meaning we have to check if te now is in the next s interval
		if sb <= tb && se <= te {
			//log.Println("s overlaps t")
			tt, tb, te = nil, sb, te // now [tb, te) is the current candidate, but it is not tt
			ss, s, sb, se = s.headx()
			continue
		}

		// [tb ...[sb .. te)...se), meaning we have to check if se now is in the next t interval
		if tb <= sb && te <= se {
			//log.Println("t overlaps s")
			ss, sb, se = nil, tb, se // now [sb, se) is the current candidate, but it is not ss
			tt, t, tb, te = t.headx()
			continue
		}

		panic("impossible interval order")
	}

	if sb != se { // we have a pending [sb, se), and perhaps some more s
		if ss == nil { // make new cover for [sb, se)
			r = append(r, interval(sb, se)...)
		} else {
			r = append(r, ss...)
		}
		r = append(r, s...) // [tb, te) is t[:tt], or tt=0 but then we also want to append the rest
	}

	if tb != te { // we have a pending [tb, te), and perhaps some more t
		if tt == nil { // make new cover for [tb, te)
			r = append(r, interval(tb, te)...)
		} else {
			r = append(r, tt...)
		}
		r = append(r, t...) // [tb, te) is t[:tt], or tt=0 but then we also want to append the rest
	}

	return r
}

// Intersection returns a Set63 containing all elements of s that are also in t.
func (s Set63) Intersection(t Set63) Set63 {
	if t.IsEmpty() {
		return t
	}
	if s.IsEmpty() {
		return s
	}
	m := len(s)
	if m > len(t) {
		m = len(t)
	}
	r := make(Set63, 0, m) // reasonable underestimate

	ss, s, sb, se := s.headx() // s[:ss] is the first contiguous interval in s
	tt, t, tb, te := t.headx() // t[:tt] is the first contiguous interval in t

	for sb != se && tb != te {
		// 6 possibilities:

		// s is entirely contained in t
		if tb <= sb && se <= te {
			if ss != nil {
				r = append(r, ss...)
			} else {
				r = append(r, interval(sb, se)...)
			}
			ss, s, sb, se = s.headx()
			continue
		}

		// t is entirely contained in s
		if sb <= tb && te <= se {
			if tt != nil {
				r = append(r, tt...)
			} else {
				r = append(r, interval(tb, te)...)
			}
			tt, t, tb, te = t.headx()
			continue
		}

		// s comes entirely before t
		if se <= tb {
			ss, s, sb, se = s.headx()
			continue
		}

		// t comes entirely before s
		if te <= sb {
			tt, t, tb, te = t.headx()
			continue
		}

		// [sb ...[tb .. se)...te), meaning we can insert [tb ..se), but we should keep t as [se, te) for further checks
		if sb <= tb && se <= te {
			//log.Println("s overlaps t")
			r = append(r, interval(tb, se)...)
			tt, tb, te = nil, se, te // now [tb, te) is the current candidate, but it is not tt
			ss, s, sb, se = s.headx()
			continue
		}

		// [tb ...[sb .. te)...se),
		if tb <= sb && te <= se {
			//log.Println("t overlaps s")
			r = append(r, interval(sb, te)...)
			ss, sb, se = nil, te, se // now [sb, se) is the current candidate, but it is not ss
			tt, t, tb, te = t.headx()
			continue
		}

		panic("impossible interval order")
	}

	return r
}

// Intersects returns whether s and t have any element in common.
func (s Set63) Intersects(t Set63) bool {
	for len(s) > 0 && len(t) > 0 {
		for len(s) <= len(t) {
			i := t.search(0, len(t), s[0])
			if t[i].contains(s[0]) || s[0].contains(t[i]) {
				return true
			}
			t, s = t[i:], s[1:]
		}
		if len(s) == 0 {
			break
		}
		for len(t) <= len(s) {
			i := s.search(0, len(s), t[0])
			if t[0].contains(s[i]) || s[i].contains(t[0]) {
				return true
			}
			t, s = t[1:], s[i:]
		}
	}
	return false
}

// Complement returns a Set63 containing all elements of [0...1<<63) that are not in s.
func (s Set63) Complement() (r Set63) {
	r = make(Set63, 0, len(s)+1) // reasonable estimate. worst case blowup is 63x
	b := uint64(0)
	_, s, sb, se := s.headx()
	for sb != se && b < 1<<63 {
		r = append(r, interval(b, sb)...)
		b = se
		_, s, sb, se = s.headx()

	}
	if b < 1<<63 {
		r = append(r, interval(b, 1<<63)...)
	}
	return r
}

// Search finds the position in s that c should be in, bracketed by i and j.
func (s Set63) search(i, j int, c cell63) int {
	for i < j {
		h := i + (j-i)/2
		// i ≤ h < j
		if s[h] < c {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// Contains returns whether elem is in s.
func (s Set63) Contains(elem int64) bool {
	if elem >= 0 {
		i := s.search(0, len(s), singleton(elem))
		if i > 0 && uint64(elem) < s[i-1].end() {
			return true
		}
		if i < len(s) && s[i].begin() <= uint64(elem) {
			return true
		}
	}
	return false
}

// Interval returns a Set63 that covers the contiguous closed interval [min, max].
// If min > max returns nil.
func Interval(min, max int64) Set63 {
	if min < 0 {
		panic("Set63 can not contain negative elements")
	}
	if min > max {
		return nil
	}
	return interval(uint64(min), uint64(max)+1)
}

func interval(min, max uint64) (s Set63) {
	// reserve expected number of cells needed
	s = make(Set63, 0, cell63(max-min).level())
	for min < max {
		c := singleton(int64(min))
		p := c.parent()
		for p != 0 && p.begin() == min && p.end() <= max {
			c = p
			p = c.parent()
		}
		s = append(s, c)
		min = c.end()
	}
	return
}

// IsEmpty returns whether the set s contains no elements.
func (s Set63) IsEmpty() bool { return len(s) == 0 }

// Equal returns whether the set s has the same elements as t.
func (s Set63) Equal(t Set63) bool {
	if len(s) != len(t) {
		return false
	}
	for i := range s {
		if s[i] != t[i] {
			return false
		}
	}
	return true
}

// Count returns the number of elements in s.
func (s Set63) Count() (n uint64) {
	for _, v := range s {
		n += uint64(v.lsb())
	}
	return
}

// Span returns the smallest and the largest element inclusive of s, or 0,-1 if s is empty.
func (s Set63) Span() (begin, end int64) {
	if len(s) == 0 {
		return 0, -1
	}
	return int64(s[0].begin()), int64(s[len(s)-1].end() - 1)
}

// ForEach calls the function f with each element of s until f returns false.
func (s Set63) ForEach(f func(int64) bool) {
	for _, v := range s {
		for i := v.begin(); i < v.end(); i++ {
			if !f(int64(i)) {
				return
			}
		}
	}
}

// ForEachInterval calls the function f with each contiguous closed interval [b, e]  s until f returns false.
func (s Set63) ForEachInterval(f func(b, e int64) bool) {
	_, s, sb, se := s.headx()
	for sb != se {
		if !f(int64(sb), int64(se - 1)) {
			return
		}
		_, s, sb, se = s.headx()
	}
}

// String returns "[begin, end)" for cells.  Useful in debugging, e.g. fmt.Println([]cell63(s)).
func (c cell63) String() string { return fmt.Sprintf("[%d, %d)", c.begin(), c.end()) }

// String returns "∅" for the empty set or "[b, e) ∪ ... [b', e'-1]" for non empty ones.
func (s Set63) String() string {
	if len(s) == 0 {
		return "∅"
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "[%d, ", s[0].begin())
	for i := 1; i < len(s); i++ {
		if s[i-1].end() == s[i].begin() {
			continue
		}
		fmt.Fprintf(&b, "%d) ∪ [%d, ", s[i-1].end(), s[i].begin())
	}
	fmt.Fprintf(&b, "%d]", s[len(s)-1].end()-1)

	return b.String()
}

// unnormalized returns the cells of s that shouldn't be there.
func (s Set63) unnormalized() (r []cell63) {
	for i := 1; i < len(s); i++ {
		if s[i-1] >= s[i] || s[i].contains(s[i-1]) || s[i-1].contains(s[i]) || s[i].sibling() == s[i-1] {
			r = append(r, s[i])
		}
	}
	return
}


// int64Slice implements sort.Interface
type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
