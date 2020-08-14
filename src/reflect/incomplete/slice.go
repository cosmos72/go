// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type iSliceType struct {
	elem Type
}

var rtypeSlice *rtype = unwrap(reflect.TypeOf(make([]unsafe.Pointer, 0)))

// SliceOf is analogous to reflect.SliceOf.
func SliceOf(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.SliceOf(ielem.complete))
	}
	incomplete := *rtypeSlice
	return &itype{
		named:      nil,
		incomplete: &incomplete,
		comparable: tfalse,
		iflag:      iflagSize,
		info: iSliceType{
			elem: elem,
		},
	}
}

func (info iSliceType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "[]"...)
	return info.elem.printTo(dst, "")
}

func (info iSliceType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// slices always have known, fixed size
	return true
}

func (info iSliceType) computeHashStr(t *itype) {
	ielem := info.elem.(*itype)
	computeHashStr(ielem)

	// Make a slice type.
	var islice interface{} = ([]unsafe.Pointer)(nil)
	prototype := *(**sliceType)(unsafe.Pointer(&islice))
	slice := *prototype
	slice.tflag = 0
	s := t.string()
	slice.str = resolveReflectName(newName(s, "", false))
	slice.hash = fnv1(ielem.incomplete.hash, '[')
	slice.ptrToThis = 0

	// TODO canonicalize ielem.incomplete and t.incomplete
	slice.elem = ielem.incomplete
	t.incomplete = &slice.rtype
}

func (info iSliceType) completeType(t *itype) {
	panic("unimplemented")
}
