// Copyright 2013 Luuk van Dijk. All Rights Reserved.
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

/*
 WORK IN PROGRESS


 gobdump reads a gob on stdin and dumps types and/or values in a readable form.
 gobdump -x produces a small gob on stdout for testing.
*/
package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

// Structures for test output
type TestBase [5]int

type TestNest map[int]string

type TestStruct struct {
	TestBase
	A uint
	B int
	C []TestNest
}

var testData = &TestStruct{TestBase: [5]int{5, 4, 3, 2, 1}, A: 15, B: -3, C: []TestNest{map[int]string{42: "life", 53: "blue"}}}

type limitedByteReader struct {
	r   io.ByteReader
	lim uint64
}

func (l *limitedByteReader) ReadByte() (byte, error) {
	if l.lim == 0 {
		return 0, io.EOF
	}
	l.lim--
	c, err := l.r.ReadByte()
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return c, err
}

func (l *limitedByteReader) Drain() error {
	for ; l.lim > 0; l.lim-- {
		if _, err := l.r.ReadByte(); err != nil {
			return err
		}
	}
	return nil
}

var errBadUint = errors.New("gob: encoded unsigned integer out of range")

func decodeUint(r io.ByteReader) (x uint64, err error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b <= 0x7f {
		return uint64(b), nil
	}
	n := -int(int8(b))
	if n > 8 {
		return 0, errBadUint
	}
	for ; n > 0; n-- {
		b, err = r.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
		x = x<<8 | uint64(b)
	}
	return x, nil
}

func decodeInt(r io.ByteReader) (x int64, err error) {
	xx, err := decodeUint(r)
	if err != nil {
		return 0, err
	}
	if xx&1 != 0 {
		return ^int64(xx >> 1), nil
	}
	return int64(xx >> 1), nil
}

func decodeString(r io.ByteReader) (string, error) {
	l, err := decodeUint(r)
	if err != nil {
		return "", err
	}
	log.Print("Read str len: ", l)
	var buf bytes.Buffer
	for ; l > 0; l-- {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		buf.WriteByte(b)
	}
	log.Print("Read str: ", buf.String())
	return buf.String(), nil
}

type decoder interface {
	Name() string
	String() string
}

type leafType string

func (s leafType) String() string { return string(s) }
func (s leafType) Name() string   { return string(s) }

type typeId int32

const (
	kNoType typeId = iota
	kBoolType
	kIntType
	kUintType
	kFloatType
	kByteSliceType
	kStringType
	kComplexType
	kInterfaceType
	kNoType9
	kNoType10
	kNoType11
	kNoType12
	kNoType13
	kNoType14
	kNoType15
	kWireType
	kArrayType
	kCommonType
	kSliceType
	kStructType
	kFieldType
	kSliceOfFieldType
	kMapType
)

type wireType struct {
	ArrayT  *arrayType
	SliceT  *sliceType
	StructT *structType
	MapT    *mapType
}

func (t *wireType) String() string {
	switch {
	case t.ArrayT != nil:
		return t.ArrayT.String()
	case t.SliceT != nil:
		return t.SliceT.String()
	case t.StructT != nil:
		return t.StructT.String()
	case t.MapT != nil:
		return t.MapT.String()
	}
	return "<invalid>"
}

func (t *wireType) Name() string {
	switch {
	case t.ArrayT != nil:
		return t.ArrayT.Name
	case t.SliceT != nil:
		return t.SliceT.Name
	case t.StructT != nil:
		return t.StructT.Name
	case t.MapT != nil:
		return t.MapT.Name
	}
	return "<noname>"
}

type commonType struct {
	Name string
	Id   typeId
}

type arrayType struct {
	commonType
	Elem typeId
	Len  int
}

func (t *arrayType) String() string {
	return fmt.Sprintf("type %s [%d]%s\n", t.Name, t.Len, descriptors[t.Elem].Name())
}

type sliceType struct {
	commonType
	Elem typeId
}

func (t *sliceType) String() string {
	return fmt.Sprintf("type %s []%s\n", t.Name, descriptors[t.Elem].Name())
}

type structType struct {
	commonType
	Field []*fieldType
}

func (t *structType) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "type %s struct {\n", t.Name)
	for _, f := range t.Field {
		fmt.Fprintf(&b, "\t%s\t%s\n", f.Name, descriptors[f.Id].Name())
	}
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

type fieldType struct {
	Name string
	Id   typeId
}

type mapType struct {
	commonType
	Key  typeId
	Elem typeId
}

func (t *mapType) String() string {
	return fmt.Sprintf("type %s map[%s]%s\n", t.Name, descriptors[t.Key].Name(), descriptors[t.Elem].Name())
}

var descriptors = map[typeId]decoder{
	kBoolType:      leafType("bool"),
	kIntType:       leafType("int"),
	kUintType:      leafType("uint"),
	kFloatType:     leafType("float64"),
	kByteSliceType: leafType("[]byte"),
	kStringType:    leafType("string"),
	kComplexType:   leafType("complex128"),
	kWireType: &wireType{
		StructT: &structType{
			commonType{"wireType", kWireType},
			[]*fieldType{
				{"ArrayT", kArrayType},
				{"SliceT", kSliceType},
				{"StructT", kStructType},
				{"MapT", kMapType},
			}}},
	kCommonType: &wireType{
		StructT: &structType{
			commonType{"commonType", kCommonType},
			[]*fieldType{
				{"Name", kStringType},
				{"Id", kIntType},
			}}},
	kArrayType: &wireType{
		StructT: &structType{
			commonType{"arrayType", kArrayType},
			[]*fieldType{
				{"commonType", kCommonType},
				{"Elem", kIntType},
				{"Len", kIntType},
			}}},
	kSliceType: &wireType{
		StructT: &structType{
			commonType{"sliceType", kSliceType},
			[]*fieldType{
				{"commonType", kCommonType},
				{"Elem", kIntType},
			}}},
	kMapType: &wireType{
		StructT: &structType{
			commonType{"mapType", kMapType},
			[]*fieldType{
				{"commonType", kCommonType},
				{"Key", kIntType},
				{"Elem", kIntType},
			}}},
	kFieldType: &wireType{
		StructT: &structType{
			commonType{"fieldType", kFieldType},
			[]*fieldType{
				{"Name", kStringType},
				{"Id", kIntType},
			}}},
	kSliceOfFieldType: &wireType{
		SliceT: &sliceType{
			commonType{"sliceType", kSliceType},
			kFieldType,
		}},
	kStructType: &wireType{
		StructT: &structType{
			commonType{"structType", kStructType},
			[]*fieldType{
				{"commonType", kCommonType},
				{"Fields", kSliceOfFieldType},
			}}},
}

// decode according to the structure
func decodeWireType(r io.ByteReader) *wireType {
	wt := new(wireType)
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			wt.ArrayT = decodeArrayType(r)
		case 1:
			wt.SliceT = decodeSliceType(r)
		case 2:
			wt.StructT = decodeStructType(r)
		case 3:
			wt.MapT = decodeMapType(r)
		}
	}
	return wt
}

func decodeCommonType(r io.ByteReader) (name string, id typeId) {
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			name, _ = decodeString(r)
		case 1:
			i, _ := decodeInt(r)
			id = typeId(i)
		}
	}
	return name, id
}

func decodeArrayType(r io.ByteReader) *arrayType {
	at := new(arrayType)
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			at.Name, at.Id = decodeCommonType(r)
		case 1:
			nn, _ := decodeInt(r)
			at.Elem = typeId(nn)
		case 2:
			nn, _ := decodeInt(r)
			at.Len = int(nn)
		}
	}
	return at
}

func decodeSliceType(r io.ByteReader) *sliceType {
	at := new(sliceType)
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			at.Name, at.Id = decodeCommonType(r)
		case 1:
			nn, _ := decodeInt(r)
			at.Elem = typeId(nn)
		}
	}
	return at
}

func decodeStructType(r io.ByteReader) *structType {
	at := new(structType)
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			at.Name, at.Id = decodeCommonType(r)
		case 1:
			fcnt, _ := decodeUint(r)
			at.Field = make([]*fieldType, fcnt)
			for i := 0; i < int(fcnt); i++ {
				name, id := decodeCommonType(r)
				at.Field[i] = &fieldType{name, id}
			}
		}
	}
	return at
}

func decodeMapType(r io.ByteReader) *mapType {
	at := new(mapType)
	f := -1
	for {
		df, err := decodeUint(r)
		if err != nil {
			log.Fatal(err)
		}
		if df == 0 {
			break
		}
		f += int(df)
		switch f {
		case 0:
			at.Name, at.Id = decodeCommonType(r)
		case 1:
			nn, _ := decodeInt(r)
			at.Key = typeId(nn)
		case 2:
			nn, _ := decodeInt(r)
			at.Elem = typeId(nn)
		}
	}
	return at
}

var (
	xflg = flag.Bool("x", false, "Dump test gob on stdout and exit")
)

func main() {
	flag.Parse()

	if *xflg {
		if err := gob.NewEncoder(os.Stdout).Encode(testData); err != nil {
			log.Fatal(err)
		}
		return
	}

	r := bufio.NewReader(os.Stdin)
	for {

		n, err := decodeUint(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rr := &limitedByteReader{r, n}

		log.Print("Record of ", n, " bytes")

		tp, err := decodeInt(rr)
		if err != nil {
			log.Print(err)
		}

		if tp < 0 {
			log.Print("Defining typeid ", -tp)
			wt := decodeWireType(rr)
			descriptors[typeId(-tp)] = wt
		} else {
			log.Print("Value of type ", tp)
		}

		if rr.lim > 0 {
			log.Print("Skipping ", rr.lim, " bytes")
			rr.Drain()
		}

	}


	for k, v := range descriptors {
		if k > 32 {
			fmt.Println(k, v)
		}
	}

}
