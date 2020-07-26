// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestOf(t *testing.T) {
	vals := []interface{}{
		false,
		int(0), int8(0), int16(0), int32(0), int64(0),
		uint(0), uint8(0), uint16(0), uint32(0), uint64(0), uintptr(0),
		float32(0), float64(0), complex64(0), complex128(0),
		[0]int{}, make(chan int), func(...int) {}, map[int]int{}, new(int),
		[](*int){}, "", struct{}{}, unsafe.Pointer(nil),
	}
	for x := range vals {
		rt := reflect.TypeOf(x)
		actual := Of(rt)
		expected := &itype{
			named:    &namedType{name: rt.Name(), pkgPath: rt.PkgPath()},
			iflag:    iflagSize,
			complete: rt,
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n\texpected\t%+v\n\tactual\t%+v", expected, actual)
		}
	}
}

type dummy struct{}

func (d dummy) String() string {
	return "dummy"
}

func TestOfWithMethods(t *testing.T) {
	x := dummy{}
	rt := reflect.TypeOf(x)
	actual := Of(rt)
	expected := &itype{
		named: &namedType{name: rt.Name(), pkgPath: rt.PkgPath()},
		methods: &[]Method{
			Method{
				Name:    "String",
				PkgPath: "",
				Type: &itype{
					iflag:    iflagSize,
					complete: reflect.TypeOf(dummy.String),
				},
			},
		},
		iflag:    iflagSize,
		complete: rt,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\n\texpected\t%+v\n\tactual\t%+v", expected, actual)
	}
}
