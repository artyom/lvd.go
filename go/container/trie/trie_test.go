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

package trie

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"
)

func TestThatItWorks(t *testing.T) {

	tc := []string{
		"aardvark",
		"abro",
		"abrocome",
		"abrogable",
		"abrogate",
		"abrogation",
		"abrogative",
		"abrogator",
		"abronah",
		"abroniaaaaa",
		"abroniaaaab",
		"abroniaaa",
	}

	m := make(map[string]string, len(tc))
	for _, s := range tc {
		m[s] = s
	}

	var tr Trie
	for _, s := range m { // getting them from m gives a randomized order
		tr.Put(s, s)
	}

	//	tr.dump(1)
	//	t.Error(tr.shape())

	// We can retrieve what we put in 
	for _, s := range tc {
		if v, ok := tr.Get(s).(string); !ok || v != s {
			if ok {
				t.Error("tr[", s, "] == ", v, ", expecting ", s)
			} else {
				t.Error("tr[", s, "] == nil, expecting ", s)
			}
		}
	}

	// we don't retrieve any prefixes (except explicitly inserted ones)
	for _, s := range tc {
		for i := 0; i < len(s)-1; i++ {
			if _, ok := m[s[:i]]; ok {
				continue
			}
			if v := tr.Get(s[:i]); v != nil {
				t.Error("tr[", s[:i], "] == ", v, ", expecting nil")
			}
		}
	}

	// All reproduces them all in sorted order
	prev := ""
	for kv := range tr.All() {
		s := kv.Key
		if v, ok := kv.Value.(string); !ok || v != s {
			if ok {
				t.Error("tr[", s, "] == ", v, ", expecting ", s)
			} else {
				t.Error("tr[", s, "] == nil, expecting ", s)
			}
		}

		if _, ok := m[s]; !ok {
			t.Error("tr[", s, "] == ", kv.Value, ", but should not exist")
		}

		if prev >= kv.Key {
			t.Errorf("out of order element: %+v after %+v", kv.Key, prev)
		}
		prev = kv.Key

		delete(m, kv.Key)
	}

	// All exhausts
	if len(m) > 0 {
		t.Error("Unretrieved: ", m)
	}
}

// Benchmarks to compare inserting random strings into a map or a trie and retrieving them in sorted order
// generate 10000 strings from a limited alphabet (8 characters) to get a fair probability of shared prefixes.
const alphabet = 8

var tc []string

func init() {
	var b bytes.Buffer
	for i := 0; i < 10000; i++ {
		b.Reset()
		for l := rand.Intn(4) + 1; l > 0; l-- {
			ch := byte(65 + rand.Intn(alphabet))
			for r := rand.Intn(4) + 1; r > 0; r-- {
				b.WriteByte(ch)
			}
		}
		tc = append(tc, b.String())
	}
}

// just insertion, no retrieval
func nativeMap(size int) {
	m := make(map[string]string, len(tc))
	for i, s := range tc {
		if i > size {
			break
		}
		m[s] = s
	}
}

// insertion and get all in sorted order
func nativeMapAndSort(size int) {
	m := make(map[string]string, len(tc))
	for i, s := range tc {
		if i > size {
			break
		}
		m[s] = s
	}
	sl := make([]string, len(m))
	for k, _ := range m {
		sl = append(sl, k)
	}
	sort.Sort(sort.StringSlice(sl))
}

// just insertion, no retrieval
func withTrie(size int) {
	var tr Trie
	for i, s := range tc {
		if i > size {
			break
		}
		tr.Put(s, s)
	}
}

// insertion and get all in sorted order
func withTrieAndAll(size int) {
	var tr Trie
	for i, s := range tc {
		if i > size {
			break
		}
		tr.Put(s, s)
	}
	for _ = range tr.All() {
	}
}

func BenchmarkNativeMap10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(10)
	}
}
func BenchmarkNativeMap100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(100)
	}
}
func BenchmarkNativeMap1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(1000)
	}
}
func BenchmarkNativeMap10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(10000)
	}
}

func BenchmarkNativeMapAndSort10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(10)
	}
}
func BenchmarkNativeMapAndSort100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(100)
	}
}
func BenchmarkNativeMapAndSort1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(1000)
	}
}
func BenchmarkNativeMapAndSort10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(10000)
	}
}

func BenchmarkWithTrie10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(10)
	}
}
func BenchmarkWithTrie100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(100)
	}
}
func BenchmarkWithTrie1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(1000)
	}
}
func BenchmarkWithTrie10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(10000)
	}
}

func BenchmarkWithTrieAndAll10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(10)
	}
}
func BenchmarkWithTrieAndAll100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(100)
	}
}
func BenchmarkWithTrieAndAll1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(1000)
	}
}
func BenchmarkWithTrieAndAll10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(10000)
	}
}
