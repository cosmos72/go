// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"testing"
	"unsafe"
)

func compare(t *testing.T, actual Type, expected Type) {
	if reflect.DeepEqual(actual, expected) {
		return
	}
	t.Errorf("\n\texpected\t%+v\n\tactual\t%+v", expected, actual)
}

var values = []interface{}{
	false,
	int(0), int8(0), int16(0), int32(0), int64(0),
	uint(0), uint8(0), uint16(0), uint32(0), uint64(0), uintptr(0),
	float32(0), float64(0), complex64(0), complex128(0),
	[0]int{}, make(chan int), func(...int) {}, map[int]int{}, new(int),
	[](*int){}, "", struct{}{}, unsafe.Pointer(nil),
}

func TestArrayOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := ArrayOf(1, Of(rt))
		expected := &itype{
			comparable: makeTribool(rt.Comparable()),
			iflag:      iflagSize,
			complete:   reflect.ArrayOf(1, rt),
		}
		compare(t, actual, expected)
	}
}

func TestChanOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := ChanOf(reflect.BothDir, Of(rt))
		expected := &itype{
			comparable: ttrue,
			iflag:      iflagSize,
			complete:   reflect.ChanOf(reflect.BothDir, rt),
		}
		compare(t, actual, expected)
	}
}

func TestFuncOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		rtslice := []reflect.Type{rt}
		it := Of(rt)
		itslice := []Type{it}
		actual := FuncOf(itslice, itslice, false)
		expected := &itype{
			comparable: tfalse,
			iflag:      iflagSize,
			complete:   reflect.FuncOf(rtslice, rtslice, false),
		}
		compare(t, actual, expected)
	}
}

func TestInterfaceOf(t *testing.T) {
	actual := InterfaceOf(nil)
	expected := &itype{
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &rtype{
			size:       rtypeInterface.size,
			align:      rtypeInterface.align,
			fieldAlign: rtypeInterface.fieldAlign,
			kind:       kInterface,
		},
		info: iInterfaceType{},
	}
	compare(t, actual, expected)
}

func TestMapOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		if !rt.Comparable() {
			continue
		}
		it := Of(rt)
		actual := MapOf(it, it)
		expected := &itype{
			comparable: tfalse,
			iflag:      iflagSize,
			complete:   reflect.MapOf(rt, rt),
		}
		compare(t, actual, expected)
	}
}

func TestNamedOf(t *testing.T) {
	name, pkgPath := "foo", "my/pkg/path"
	actual := NamedOf(name, pkgPath)
	expected := &itype{
		named: &namedType{
			name:    name,
			pkgPath: pkgPath,
			str:     filename(pkgPath) + "." + name,
		},
		comparable: tunknown,
		iflag:      0,
	}
	compare(t, actual, expected)
}

func TestOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := Of(rt)
		var named *namedType
		if rt.Name() != "" {
			named = &namedType{
				name:    rt.Name(),
				pkgPath: rt.PkgPath(),
				str:     rt.String(),
			}
		}
		expected := &itype{
			named:      named,
			comparable: makeTribool(rt.Comparable()),
			iflag:      iflagSize,
			complete:   rt,
		}
		compare(t, actual, expected)
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
		named: &namedType{
			name:    rt.Name(),
			pkgPath: rt.PkgPath(),
			str:     filename(rt.PkgPath()) + "." + rt.Name(),
		},
		method: &[]Method{
			Method{
				Name:    "String",
				PkgPath: "",
				Type: &itype{
					comparable: tfalse,
					iflag:      iflagSize,
					complete:   reflect.TypeOf(dummy.String),
				},
			},
		},
		comparable: makeTribool(rt.Comparable()),
		iflag:      iflagSize,
		complete:   rt,
	}
	compare(t, actual, expected)
}

func TestPtrTo(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := PtrTo(Of(rt))
		expected := &itype{
			comparable: ttrue,
			iflag:      iflagSize,
			complete:   reflect.PtrTo(rt),
		}
		compare(t, actual, expected)
	}
}

func TestSliceOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := SliceOf(Of(rt))
		expected := &itype{
			comparable: tfalse,
			iflag:      iflagSize,
			complete:   reflect.SliceOf(rt),
		}
		compare(t, actual, expected)
	}
}

func TestStructOf(t *testing.T) {
	fieldrt := reflect.TypeOf(int(0))
	fieldt := Of(fieldrt)
	actual := StructOf([]StructField{
		{Name: "First", Type: fieldt},
		{Name: "Second", Type: fieldt},
	})
	rt := reflect.StructOf([]reflect.StructField{
		{Name: "First", Type: fieldrt},
		{Name: "Second", Type: fieldrt},
	})
	expected := &itype{
		comparable: makeTribool(fieldrt.Comparable()),
		iflag:      iflagSize,
		complete:   rt,
	}
	compare(t, actual, expected)
}
