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

// SliceOf is analogous to reflect.SliceOf.
func SliceOf(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.SliceOf(ielem.complete))
	}
	// Make a slice type.
	var islice interface{} = ([]unsafe.Pointer)(nil)
	slice := **(**sliceType)(unsafe.Pointer(&islice))
	slice.tflag = 0
	slice.ptrToThis = 0
	slice.elem = nil

	// TODO canonicalize return value
	return &itype{
		named:      nil,
		incomplete: &slice.rtype,
		comparable: tfalse,
		iflag:      iflagSize,
		info: &iSliceType{
			elem: elem,
		},
	}
}

func (info *iSliceType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "[]"...)
	return info.elem.printTo(dst, "")
}

func (info *iSliceType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// slices always have known, fixed size
	return true
}

func (info *iSliceType) computeHashStr(t *itype) {
	ielem := info.elem.(*itype)
	computeHashStr(ielem)

	slice := (*sliceType)(unsafe.Pointer(&t.incomplete))
	slice.str = resolveReflectName(newName(t.string(), "", false))
	slice.hash = fnv1(ielem.incomplete.hash, '[')
	slice.elem = ielem.incomplete
}

func (info *iSliceType) completeType(t *itype) {
	panic("unimplemented")
}
