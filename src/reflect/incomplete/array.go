// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
)

type iArrayType struct {
	elem  Type
	count int
}

// ArrayOf creates an incomplete array type with the given count and
// element type described by elem.
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("incomplete.ArrayOf: element count is negative")
	}
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ArrayOf(count, ielem.complete))
	}
	return &itype{
		named:      nil,
		comparable: ielem.comparable,
		iflag:      ielem.iflag & iflagSize,
		incomplete: &rtype{
			size: uintptr(count) * ielem.size(),
			kind: kArray,
		},
		info: iArrayType{
			elem:  elem,
			count: count,
		},
	}
}

func (info iArrayType) printTo(dst []byte, sep string) []byte {
	dst = append(append(append(append(
		dst, sep...), '['), strconv.Itoa(info.count)...), ']')
	return info.elem.printTo(dst, "")
}

func (info iArrayType) computeSize(t *itype, work map[*itype]struct{}) {
	ielem := info.elem.(*itype)
	computeSize(ielem, work)
	if ielem.iflag&iflagSize == 0 {
		return
	}
	esize := ielem.size()
	if esize > 0 {
		max := ^uintptr(0) / esize
		if uintptr(info.count) > max {
			panic("incomplete.ArrayOf: array size would exceed virtual address space")
		}
	}
	t.setSize(uintptr(info.count)*esize, ielem.align(), ielem.fieldAlign())
}

func (info iArrayType) prepareRtype(t *itype) {
	panic("unimplemented")
}

func (info iArrayType) completeType(t *itype) {
	panic("unimplemented")
}
