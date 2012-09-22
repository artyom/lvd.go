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

// Package trie implements a byte trie with edge compression.
package trie

import (
	"bytes"
	"fmt"
)

// A trie maintains a sorted collection of values keyed on a string.
///Insertion is O(len(key)). Unlike Go's built-in map there is no 
// distinction between a nil and a non-existent value.
// The zero value for Trie is an empty trie ready to use.
type Trie struct {
	suffix   string
	value    interface{}
	children []Trie
	base     byte
}

// Find the largest i such that a[:i] == b[:i]
func commonPrefix(a, b string) int {
	var i int
	for i = 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

// Put inserts or replaces a value in the trie.  To remove a value
// insert nil.
func (t *Trie) Put(key string, value interface{}) {
	if t.children == nil && t.value == nil { // empty node
		t.suffix = key
		t.value = value
		return
	}

	s := commonPrefix(t.suffix, key)

	if s < len(t.suffix) {
		// split on s: turn t into a node with suffix[:s]
		// and move the contents to child[suffix[s]-t.base] with suffix[s+1:]
		*t = Trie{t.suffix[:s], nil, []Trie{{t.suffix[s+1:], t.value, t.children, t.base}}, t.suffix[s]}
	}

	if s == len(key) {
		t.value = value
		return
	}

	if len(t.children) == 0 {
		t.children = make([]Trie, 1)
		t.base = key[s]
	} else {
		newbase := t.base
		newlen := len(t.children)
		for key[s] < newbase || int(key[s]) >= int(newbase)+newlen {
			newlen *= 2
			newbase &= ^byte(newlen - 1)
		}
		if newlen != len(t.children) {
			newch := make([]Trie, newlen)
			copy(newch[t.base-newbase:], t.children)
			t.children = newch
			t.base = newbase
		}
	}

	t.children[key[s]-t.base].Put(key[s+1:], value)

	return
}

// Get retrieves an element from the trie if it exists, or nil if it does not.
func (t *Trie) Get(key string) interface{} {
	s := commonPrefix(t.suffix, key)

	if s < len(t.suffix) {
		return nil
	}

	if s == len(key) {
		return t.value
	}

	if key[s] < t.base || int(key[s]) > int(t.base)+len(t.children) {
		return nil
	}

	return t.children[key[s]-t.base].Get(key[s+1:])
}

func (t *Trie) forEach(f func(string, interface{}) bool, buf *bytes.Buffer) bool {
	if t.value != nil || t.children != nil {

		pfx := buf.Len()
		buf.WriteString(t.suffix)

		if t.value != nil && !f(buf.String(), t.value) {
			return false
		}

		if t.children != nil {
			l := buf.Len()
			buf.WriteByte(t.base)
			for _, child := range t.children {
				if !child.forEach(f, buf) {
					return false
				}
				buf.Bytes()[l]++
			}
		}

		buf.Truncate(pfx)

	}

	return true
}

// ForEach will apply the function f to each key, value pair in the
// Trie in sorted (depth-first pre-)order.  if f returns false, the
// iteration will stop.
func (t *Trie) ForEach(f func(string, interface{}) bool) {
	var buf bytes.Buffer
	t.forEach(f, &buf)
}

// String returns a multiline string representation of the trie
// in the form 
//    trie[
//       key1: value1
//       key2: value2
//       ....
//    ]
func (t *Trie) String() string {
	var buf bytes.Buffer
	buf.WriteString("trie{\n")
	t.ForEach(func(key string, val interface{}) bool {
		fmt.Fprintf(&buf, "\t%s:%v\n", key, val)
		return true
	})
	buf.WriteString("}")
	return buf.String()
}

// debug
const spaces = "                                                                                "

func (t *Trie) dump(level int) {
	if level > len(spaces) {
		level = len(spaces)
	}
	fmt.Printf("%s: %v\n", t.suffix, t.value)
	if t.children != nil {
		fmt.Printf("%s<%d>\n", spaces[:4*level], len(t.children))
	}
	c := t.base
	for _, ch := range t.children {
		if ch.value != nil || ch.children != nil {
			if c >= 32 && c < 128 {
				fmt.Printf("%s['%c']", spaces[:4*level], c)
			} else {
				fmt.Printf("%s[%d]", spaces[:4*level], c)
			}
			ch.dump(level + 1)
		}
		c++
	}
}

func (t *Trie) shape() (ln, sz int) {
	if t.value != nil {
		ln++
	}
	sz++
	for _, c := range t.children {
		l, s := c.shape()
		ln += l
		sz += s
	}
	return
}
