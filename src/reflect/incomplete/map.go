// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type iMapType struct {
	key  Type
	elem Type
}

// MapOf creates an incomplete map type with the given key and element types.
func MapOf(key, elem Type) Type {
	ikey := key.(*itype)
	ielem := elem.(*itype)
	if ikey.complete != nil && ielem.complete != nil {
		return Of(reflect.MapOf(ikey.complete, ielem.complete))
	}
	if ikey.comparable == tfalse {
		panic("incomplete.MapOf: invalid key type, is not comparable")
	}

	// Make a map type.
	var imap interface{} = (map[unsafe.Pointer]unsafe.Pointer)(nil)
	mt := **(**mapType)(unsafe.Pointer(&imap))
	mt.tflag = 0
	mt.flags = 0
	mt.ptrToThis = 0

	return &itype{
		named:      nil,
		comparable: tfalse,
		iflag:      iflagSize,
		incomplete: &mt.rtype,
		info: iMapType{
			key:  key,
			elem: elem,
		},
	}
}

func (info iMapType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "map["...)
	dst = info.key.printTo(dst, "")
	dst = append(dst, ']')
	return info.elem.printTo(dst, "")
}

func (info iMapType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// maps always have known, fixed size
	return true
}

func (info iMapType) computeHashStr(t *itype) {
	ikey := info.elem.(*itype)
	computeHashStr(ikey)
	if ikey.incomplete.equal == nil {
		panic("incomplete.Complete: invalid map key type, is not comparable: " +
			ikey.string())
	}
	ielem := info.elem.(*itype)
	computeHashStr(ielem)

	// TODO canonicalize t.incomplete, ikey.incomplete and ielem.incomplete
	prepareMapType(t.incomplete, ikey.incomplete, ielem.incomplete, t.string())
}

// Make a map type.
// Note: flag values must match those used in the TMAP case
// in ../../cmd/compile/internal/gc/reflect.go:dtypesym.
func prepareMapType(t *rtype, ktyp *rtype, etyp *rtype, str string) {
	mt := (*mapType)(unsafe.Pointer(t))
	mt.str = resolveReflectName(newName(str, "", false))
	mt.tflag = 0
	mt.hash = fnv1(etyp.hash, 'm', byte(ktyp.hash>>24), byte(ktyp.hash>>16), byte(ktyp.hash>>8), byte(ktyp.hash))
	mt.key = ktyp
	mt.elem = etyp
	mt.bucket = bucketOf(ktyp, etyp)
	mt.hasher = func(p unsafe.Pointer, seed uintptr) uintptr {
		return typehash(ktyp, p, seed)
	}
	mt.flags = 0
	if ktyp.size > maxKeySize {
		mt.keysize = uint8(ptrSize)
		mt.flags |= 1 // indirect key
	} else {
		mt.keysize = uint8(ktyp.size)
	}
	if etyp.size > maxValSize {
		mt.valuesize = uint8(ptrSize)
		mt.flags |= 2 // indirect value
	} else {
		mt.valuesize = uint8(etyp.size)
	}
	mt.bucketsize = uint16(mt.bucket.size)
	if isReflexive(ktyp) {
		mt.flags |= 4
	}
	if needKeyUpdate(ktyp) {
		mt.flags |= 8
	}
	if hashMightPanic(ktyp) {
		mt.flags |= 16
	}
	mt.ptrToThis = 0
}

func (info iMapType) completeType(t *itype) {
	panic("unimplemented")
}
