// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

var (
	RTypeOfInt reflect.Type = reflect.TypeOf((int(0)))
	rTypeOfInt *rtype       = unwrap(RTypeOfInt)
	typeOfInt  Type         = Of(RTypeOfInt)
)

func inspectNamed(t *testing.T, rt reflect.Type) {
	// fmt.Printf("--- type %s ---\n%#v\n", rt.String(), rt)
	if rt.Name() == "" || rt.PkgPath() == "" || rt.Size() == 0 {
		t.Errorf("created bad reflect.Type %v kind %s", rt, rt.Kind())
		return
	}
	/* x := */ reflect.New(rt).Elem().Interface()
	// t.Logf("created %v // %v: %T", x, rt.Kind(), x)
}

var values = []interface{}{
	false,
	int(0), int8(0), int16(0), int32(0), int64(0),
	uint(0), uint8(0), uint16(0), uint32(0), uint64(0), uintptr(0),
	float32(0), float64(0), complex64(0), complex128(0),
	[1]int{}, make(chan int), func(...int) {}, map[int]int{}, new(int),
	[](*int){}, "", struct{ X int }{0}, unsafe.Pointer(nil),
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
	actual := InterfaceOf(nil, nil)
	expected := &itype{
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &interfaceProto.rtype,
		info:       &iInterfaceType{},
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
			qname: makeQname(name, pkgPath),
		},
	}
	compare(t, actual, expected)

	// ==================================
	actual.Define(typeOfInt)
	expected = &itype{
		named: expected.named,
		iflag: iflagDefined | iflagSize,
		incomplete: &rtype{
			size:       RTypeOfInt.Size(),
			tflag:      tflagUncommon | tflagNamed | tflagRegularMemory,
			align:      uint8(RTypeOfInt.Align()),
			fieldAlign: uint8(RTypeOfInt.FieldAlign()),
			kind:       kInt,
			equal:      rTypeOfInt.equal,
			gcdata:     rTypeOfInt.gcdata,
		},
		info: typeOfInt.(*itype),
	}
	compare(t, actual, expected)

	// ==================================
	computeHashStr(actual.(*itype))
	expected.incomplete.hash = actual.(*itype).incomplete.hash
	expected.incomplete.str = actual.(*itype).incomplete.str
	compare(t, actual, expected)

	// ==================================
	completeType(actual.(*itype))
	expected.complete = wrap(expected.incomplete)
	compare(t, actual, expected)
}

func TestCompleteNamedOf(t *testing.T) {
	for i, x := range values {
		name, pkgPath := "foo"+strconv.Itoa(i), "my/pkg/path"
		named := NamedOf(name, pkgPath)
		named.Define(Of(reflect.TypeOf(x)))
		rt := Complete([]Type{named}, nil)[0]
		inspectNamed(t, rt)
	}
}

func TestOf(t *testing.T) {
	for _, x := range values {
		rt := reflect.TypeOf(x)
		actual := Of(rt)
		var named *namedType
		if rt.Name() != "" {
			named = &namedType{
				qname: qname{
					name:    rt.Name(),
					pkgPath: rt.PkgPath(),
					str:     rt.String(),
				},
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
			qname: qname{
				name:    rt.Name(),
				pkgPath: rt.PkgPath(),
				str:     filename(rt.PkgPath()) + "." + rt.Name(),
			},
			vmethod: []Method{
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
			pmethod: []Method{
				Method{
					Name:    "String",
					PkgPath: "",
					Type: &itype{
						comparable: tfalse,
						iflag:      iflagSize,
						complete:   reflect.TypeOf((*dummy).String),
					},
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

func TestPtrToNamed(t *testing.T) {
	name, pkgPath := "foo", "my/pkg/path"
	elem := NamedOf(name, pkgPath)
	actual := PtrTo(elem)
	expected := &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &rtype{
			size:       ptrSize,
			ptrdata:    ptrSize,
			tflag:      tflagRegularMemory,
			align:      ptrSize,
			fieldAlign: ptrSize,
			kind:       kPtr | kindDirectIface,
			equal:      actual.(*itype).incomplete.equal,
			gcdata:     actual.(*itype).incomplete.gcdata,
		},
		info: &iPtrType{
			elem: elem,
		},
	}
	compare(t, actual, expected)
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

func TestSliceOfNamed(t *testing.T) {
	name, pkgPath := "foo", "my/pkg/path"
	elem := NamedOf(name, pkgPath)

	actual := SliceOf(elem)
	expected := &itype{
		comparable: tfalse,
		iflag:      iflagSize,
		incomplete: &rtype{
			size:       3 * ptrSize,
			ptrdata:    ptrSize,
			tflag:      0,
			align:      ptrSize,
			fieldAlign: ptrSize,
			kind:       kSlice,
			equal:      actual.(*itype).incomplete.equal,
			gcdata:     actual.(*itype).incomplete.gcdata,
		},
		info: &iSliceType{
			elem: elem,
		},
	}
	compare(t, actual, expected)

	elem.Define(typeOfInt)
	compare(t, actual, expected)

	// computeHashStr(actual.(*itype))
	// compare(t, actual, expected) // currently panics
}

func TestStructOf(t *testing.T) {
	fieldrt := RTypeOfInt
	fieldt := typeOfInt
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

func TestStructOfNamed(t *testing.T) {
	name, pkgPath := "bar", "my/pkg/path"
	fieldt := NamedOf(name, pkgPath)
	actual := StructOf([]StructField{
		{Name: "First", Type: fieldt},
		{Name: "Second", Type: fieldt},
	})
	expected := &itype{
		incomplete: &rtype{
			hash: actual.(*itype).incomplete.hash,
			kind: kStruct,
		},
		info: &iStructType{
			[]StructField{
				{Name: "First", Type: fieldt},
				{Name: "Second", Type: fieldt},
			},
		},
	}
	compare(t, actual, expected)
}
