// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	"testing"
)

func compare(t *testing.T, actual Type, expected Type) {
	compareType(t, "", actual, expected)
}

func compareType(t *testing.T, path string, a Type, e Type) {
	if a == e {
		return
	}
	compareIType(t, path, a.(*itype), e.(*itype))
}

func compareIType(t *testing.T, path string, a *itype, e *itype) {
	if a == e {
		return
	}
	if a == nil || e == nil || a.comparable != e.comparable || a.iflag != e.iflag {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
	if a != nil && e != nil {
		compareNamed(t, path+".named", a.named, e.named)
		compareRType(t, path+".incomplete", a.incomplete, e.incomplete)
		compareReflectType(t, path+".complete", a.complete, e.complete)
		compareInfo(t, path+".info", a.info, e.info)
	}
}

func compareNamed(t *testing.T, path string, a *namedType, e *namedType) {
	if a == e {
		return
	}
	if a == nil || e == nil || a.qname != e.qname {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
	if a != nil && e != nil {
		compareMethodSlice(t, path+".vmethod", a.vmethod, e.vmethod)
		compareMethodSlice(t, path+".pmethod", a.pmethod, e.pmethod)
	}
}

func compareMethodSlice(t *testing.T, path string, a []Method, e []Method) {
	if len(a) != len(e) {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	for i := range a {
		compareMethod(t, path+"["+strconv.Itoa(i)+"]", a[i], e[i])
	}
}

func compareMethod(t *testing.T, path string, a Method, e Method) {
	if a.Name != e.Name || a.PkgPath != e.PkgPath {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
	compareType(t, path+".Type", a.Type, e.Type)
}

func compareReflectType(t *testing.T, path string, a reflect.Type, e reflect.Type) {
	if a == e {
		return
	}
	compareRType(t, path, unwrap(a), unwrap(e))
}

func compareRType(t *testing.T, path string, a *rtype, e *rtype) {
	if a == e {
		return
	}
	if a == nil || e == nil {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	if a.size != e.size || a.ptrdata != e.ptrdata ||
		a.hash != e.hash || a.tflag != e.tflag || a.align != e.align ||
		a.fieldAlign != e.fieldAlign || a.kind != e.kind ||
		(a.equal != nil) != (e.equal != nil) || a.gcdata != e.gcdata {

		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
	compareString(t, path+".String()", a.string(), e.string())
}

func compareString(t *testing.T, path string, a string, e string) {
	if a != e {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
}

func compareInfo(t *testing.T, path string, a iAnyType, e iAnyType) {
	if at, ok := a.(*itype); ok {
		a = resolveUnderlying(at)
	}
	if et, ok := e.(*itype); ok {
		e = resolveUnderlying(et)
	}
	if a == e {
		return
	}
	if a == nil || e == nil || reflect.TypeOf(a) != reflect.TypeOf(e) {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	switch a := a.(type) {
	case *iArrayType:
		compareArray(t, path, a, e.(*iArrayType))
	case *iChanType:
		compareChan(t, path, a, e.(*iChanType))
	case *iFuncType:
		compareFunc(t, path, a, e.(*iFuncType))
	case *iInterfaceType:
		compareInterface(t, path, a, e.(*iInterfaceType))
	case *iMapType:
		compareMap(t, path, a, e.(*iMapType))
	case *iPtrType:
		comparePtr(t, path, a, e.(*iPtrType))
	case *iSliceType:
		compareSlice(t, path, a, e.(*iSliceType))
	case *iStructType:
		compareStruct(t, path, a, e.(*iStructType))
	}
}

func compareArray(t *testing.T, path string, a *iArrayType, e *iArrayType) {
	if a == e {
		return
	}
	if a == nil || e == nil || a.count != e.count {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	compareType(t, path+".elem", a.elem, e.elem)
	if a.slice != nil && e.slice != nil {
		compareIType(t, path+".slice", a.slice, e.slice)
	}
}

func compareChan(t *testing.T, path string, a *iChanType, e *iChanType) {
	if a == e {
		return
	}
	if a == nil || e == nil || a.dir != e.dir {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	compareType(t, path+".elem", a.elem, e.elem)
}

func compareFunc(t *testing.T, path string, a *iFuncType, e *iFuncType) {
	if a == e {
		return
	}
	if a == nil || e == nil || a.variadic != e.variadic ||
		len(a.in) != len(e.in) || len(a.out) != len(e.out) {

		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	for i := range a.in {
		compareType(t, path+".in["+strconv.Itoa(i)+"]", a.in[i], e.in[i])
	}
	for i := range a.out {
		compareType(t, path+".out["+strconv.Itoa(i)+"]", a.out[i], e.out[i])
	}
}

func compareInterface(t *testing.T, path string,
	a *iInterfaceType, e *iInterfaceType) {

	if a == e {
		return
	}
	if a == nil || e == nil ||
		len(a.embedded) != len(e.embedded) ||
		len(a.declaredMethod) != len(e.declaredMethod) {

		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	for i := range a.embedded {
		compareType(t, path+".embedded["+strconv.Itoa(i)+"]",
			a.embedded[i], e.embedded[i])
	}
	for i := range a.declaredMethod {
		compareMethod(t, path+".declaredMethod["+strconv.Itoa(i)+"]",
			a.declaredMethod[i], e.declaredMethod[i])
	}
}

func compareMap(t *testing.T, path string, a *iMapType, e *iMapType) {
	if a == e {
		return
	}
	if a == nil || e == nil {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	compareType(t, path+".key", a.key, e.key)
	compareType(t, path+".elem", a.elem, e.elem)
}

func comparePtr(t *testing.T, path string, a *iPtrType, e *iPtrType) {
	if a == e {
		return
	}
	if a == nil || e == nil {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	compareType(t, path+".elem", a.elem, e.elem)
}

func compareSlice(t *testing.T, path string, a *iSliceType, e *iSliceType) {
	if a == e {
		return
	}
	if a == nil || e == nil {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	compareType(t, path+".elem", a.elem, e.elem)
}

func compareStruct(t *testing.T, path string, a *iStructType, e *iStructType) {
	if a == e {
		return
	}
	if a == nil || e == nil || len(a.fields) != len(e.fields) {
		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
		return
	}
	for i := range a.fields {
		compareField(t, path+".field["+strconv.Itoa(i)+"]",
			a.fields[i], e.fields[i])
	}
}

func compareField(t *testing.T, path string, a StructField, e StructField) {
	if a.Name != e.Name || a.PkgPath != e.PkgPath ||
		a.Tag != e.Tag || a.Anonymous != e.Anonymous {

		t.Errorf("mismatched %s:\n\texpected  %+v\n\tactual    %+v", path, e, a)
	}
	compareType(t, path+".Type", a.Type, e.Type)
}
