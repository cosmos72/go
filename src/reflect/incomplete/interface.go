// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"unsafe"
)

type iInterfaceType struct {
	embedded       []Type
	declaredMethod []Method
	allMethod      []Method // ordered alphabetically
}

// do NOT modify!
var interfaceProto interfaceType = makeInterfacePrototype()

func makeInterfacePrototype() interfaceType {
	var interfaceptr interface{} = (*interface{})(nil)
	ptr := *(**ptrType)(unsafe.Pointer(&interfaceptr))

	iface := *(*interfaceType)(unsafe.Pointer(ptr.elem))
	iface.hash = 0
	iface.equal = nil
	iface.str = 0
	iface.ptrToThis = 0
	iface.pkgPath.bytes = nil
	iface.methods = nil
	return iface
}

// InterfaceOf returns an incomplete interface type with the given list of
// named interface types. InterfaceOf panics if one of the given embedded types
// is unnamed or its kind is not reflect.Interface. It also panics if types
// with distinct, non-empty package paths are embedded.
func InterfaceOf(embedded []Type, method []Method) Type {

	iface := interfaceProto

	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &iface.rtype,
		info: &iInterfaceType{
			// safety: make a copy of embedded[]
			embedded: append(([]Type)(nil), embedded...),
			// safety: make a copy of method[]
			declaredMethod: append(([]Method)(nil), method...),
		},
	}
}

func (info *iInterfaceType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "interface{"...)

	if len(info.allMethod) == 0 {
		return append(dst, '}')
	}
	sep = " "
	for i := range info.allMethod {
		info.allMethod[i].printTo(dst, sep)
		sep = "; "
	}
	return append(dst, " }"...)
}

func (info *iInterfaceType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// interfaces always have known, fixed size
	return true
}

func (info *iInterfaceType) computeHashStr(t *itype) {
	panic("unimplemented")
}

func (info *iInterfaceType) completeType(t *itype) {
	panic("unimplemented")
}
