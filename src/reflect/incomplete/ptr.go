// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type iPtrType struct {
	elem Type
}

// PtrTo is analogous to reflect.PtrTo.
func PtrTo(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.PtrTo(ielem.complete))
	}
	var iptr interface{} = (*unsafe.Pointer)(nil)
	pp := **(**ptrType)(unsafe.Pointer(&iptr))
	pp.ptrToThis = 0
	pp.elem = nil

	// TODO canonicalize return value
	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &pp.rtype,
		info: &iPtrType{
			elem: elem,
		},
	}
}

func (info *iPtrType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), '*')
	return info.elem.printTo(dst, "")
}

func (info *iPtrType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// pointers always have known, fixed size
	return true
}

func (info *iPtrType) computeHashStr(t *itype) {
	ielem := info.elem.(*itype)
	computeHashStr(ielem)

	pp := (*ptrType)(unsafe.Pointer(t.incomplete))

	pp.str = resolveReflectName(newName(t.string(), "", false))
	pp.ptrToThis = 0

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	pp.hash = fnv1(ielem.incomplete.hash, '*')

	pp.elem = ielem.incomplete
}

func (info *iPtrType) completeType(t *itype) {
	panic("unimplemented")
}
