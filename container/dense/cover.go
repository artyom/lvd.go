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
	"fmt"
	"container/heap"
//	"log"
)

type cand struct {
	c cell63
	w uint64
	il, ir int
	ip int 
	g uint64
}

func (c *cand) waste() uint64 { return uint64(c.c.lsb()) - c.w }
func (c *cand) isLeaf() bool { return c.il == -1 }

// Recursively compute as much information as needed for the shaving pass
func (s Set63) shape(i, j int, c cell63, minGrain uint64, sh *[]cand) uint64 {
	if i == j {
		return 0
	}
	if uint64(c.lsb()) < minGrain*2 {  
		w := s[i:j].Count()
		*sh = append(*sh, cand{c, w, -1, -1, -1, 0})
		return w
	}

	if i + 1 == j {
		w := uint64(s[i].lsb())
		cc := s[i]
		for uint64(cc.lsb()) < minGrain {
			p := cc.parent()
			if p == 0 {
				break
			}
			cc = p
		}
		*sh = append(*sh, cand{cc, w, -1, -1, -1, 0})
		return w
	}

	cl, cr := c.children()
	m := s.search(i, j, c)

	nl := s.shape(i, m, cl, minGrain, sh)
	il := len(*sh) - 1

	nr := s.shape(m, j, cr, minGrain, sh)
	ir := len(*sh) - 1

	if nl != 0 && nr != 0 {
		// append new node and set parent and waste-gain in children
		nn := cand{c, nl+nr, il, ir, -1, 0}
		g := nn.waste() - (*sh)[il].waste() - (*sh)[ir].waste()

		(*sh)[il].ip = len(*sh)
		(*sh)[ir].ip = len(*sh)
		(*sh)[il].g = g
		(*sh)[ir].g = g
		*sh = append(*sh, nn)
	}

	return nl + nr
}

type pQ struct {
	shape []cand
	leaves []int
}

func (pq *pQ) dump() {
	for i, c := range pq.shape {
		if c.c == 0 {
			continue
		}
		if c.isLeaf() {
			fmt.Println(i, spaces[:2*c.c.level()], c.c, ":", c.w, "leaf p:", c.ip, c.g)
		} else {
			fmt.Println(i, spaces[:2*c.c.level()], c.c, ":", c.w, "( ", c.il, c.ir ,") p:", c.ip, c.g)
		}
	}
}

func (pq *pQ) getLeaves(i int, s *Set63) {
	if pq.shape[i].c != 0 && pq.shape[i].isLeaf() {
		*s = append(*s, pq.shape[i].c)
		return
	}
	pq.getLeaves(pq.shape[i].il, s)
	pq.getLeaves(pq.shape[i].ir, s)
	
}

func (pq *pQ) blowup(ip int) {
	if ip == 0 {
		panic("blowup 0")
	}
	p := &pq.shape[ip]
	pq.shape[p.il].c = 0
	pq.shape[p.ir].c = 0
	p.il, p.ir = -1, -1
}

func (pq *pQ) Push(x interface{}) { pq.leaves = append(pq.leaves, x.(int)) }
func (pq *pQ) Pop() (x interface{})   {
        l := len(pq.leaves)-1
        pq.leaves, x = pq.leaves[:l], pq.leaves[l]
        return
}

func (pq pQ) Len() int           { return len(pq.leaves) }
func (pq pQ) Swap(i, j int)      { pq.leaves[i], pq.leaves[j] = pq.leaves[j], pq.leaves[i] }
func (pq pQ) Less(i, j int) bool {
	if pq.shape[pq.leaves[i]].g != pq.shape[pq.leaves[j]].g {
		return pq.shape[pq.leaves[i]].g < pq.shape[pq.leaves[j]].g
	}
	return i < j
}

const spaces = "                                                                                                                                  "

// Cover returns a new Set63 that contains at least all elements of s,
// but does not use more than maxSize units of storage if maxSize > 0,
// and does not use intervals smaller than minGrain. If mingrain >
// 1<<62, returns the unit set [0, 1<<63)
func (s Set63) Cover(maxSize int, minGrain uint64) Set63 {
	if s.IsEmpty() {
		return Set63{}
	}

	if maxSize < 1 || len(s) < maxSize {
		maxSize = len(s)
	}

	var pq pQ
	pq.shape = make([]cand, 0, 2*len(s))
	s.shape(0, len(s), unity63, minGrain, &pq.shape)

//	pq.dump()

	pq.leaves = make([]int, 0, len(s))
	for i, c := range pq.shape {
		if c.isLeaf() {
//			log.Print(i, " is leaf", c)
			heap.Push(&pq, i)
		}
	}

	// shave: pop all leaves (in order of waste-gain)
	// if waste gain is zero blow up to parent,
	// otherwise, only if the number of leaves is larger than maxSize
	for len(pq.leaves) > 1 {
		i := heap.Pop(&pq).(int)

//		log.Print("popped ", i, ": ", pq.shape[i])
		if pq.shape[i].c == 0 {
//			log.Print(i, " already removed: ", pq.shape[i])
			continue
		}

		g := pq.shape[i].g 

		if g == 0 || len(pq.leaves) >= maxSize  {
			ip := pq.shape[i].ip
			if ip == -1 {
//				log.Print("popped root")
				return Set63{pq.shape[i].c}
			}
//			log.Print("blowing up ", ip, ": ", pq.shape[ip])
			pq.blowup(ip)
			heap.Push(&pq, ip)
			continue
		}
		
		if len(pq.leaves) < maxSize {
			break
		}
	}
	
/*
	log.Println("after shaving downto ", maxSize)
	pq.dump()
	log.Println("leaves: ", pq.leaves)
*/
	ss := make(Set63, 0, len(pq.leaves))
	pq.getLeaves(len(pq.shape)-1, &ss)
 	return ss

}
