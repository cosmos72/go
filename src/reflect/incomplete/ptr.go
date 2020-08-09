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

var rtypePtr *rtype = unwrap(reflect.TypeOf(new(unsafe.Pointer)))

// PtrTo is analogous to reflect.PtrTo.
func PtrTo(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.PtrTo(ielem.complete))
	}
	incomplete := *rtypePtr
	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &incomplete,
		info: iPtrType{
			elem: elem,
		},
	}
}

func (info iPtrType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), '*')
	return info.elem.printTo(dst, "")
}

func (info iPtrType) prepareRtype(t *itype) {
	ielem := info.elem.(*itype)
	ielem.prepareRtype(ielem)

	var iptr interface{} = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	s := t.string()
	pp.str = resolveReflectName(newName(s, "", false))
	pp.ptrToThis = 0

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	pp.hash = fnv1(ielem.incomplete.hash, '*')

	// TODO canonicalize ielem.incomplete and t.incomplete
	pp.elem = ielem.incomplete
	t.incomplete = &pp.rtype
}

func (info iPtrType) completeType(t *itype) {
	panic("unimplemented")
}
