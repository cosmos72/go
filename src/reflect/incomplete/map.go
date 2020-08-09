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

var rtypeMap *rtype = unwrap(reflect.TypeOf(make(map[unsafe.Pointer]unsafe.Pointer)))

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
	incomplete := *rtypeMap
	return &itype{
		named:      nil,
		comparable: tfalse,
		iflag:      iflagSize,
		incomplete: &incomplete,
		info: iMapType{
			key:  key,
			elem: elem,
		},
	}
}

// Make a map type.
// Note: flag values must match those used in the TMAP case
// in ../cmd/compile/internal/gc/reflect.go:dtypesym.
func makeMapType(ktyp *rtype, etyp *rtype, str string) *mapType {
	var imap interface{} = (map[unsafe.Pointer]unsafe.Pointer)(nil)
	mt := **(**mapType)(unsafe.Pointer(&imap))
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
	return &mt
}

func (info iMapType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "map["...)
	dst = info.key.printTo(dst, "")
	dst = append(dst, ']')
	return info.elem.printTo(dst, "")
}

func (info iMapType) prepareRtype(t *itype) {
	ikey := info.elem.(*itype)
	ikey.prepareRtype(ikey)
	if ikey.incomplete.equal == nil {
		panic("incomplete.Complete: invalid map key type, is not comparable: " +
			ikey.string())
	}
	ielem := info.elem.(*itype)
	ielem.prepareRtype(ielem)

	// TODO canonicalize ikey.incomplete and ielem.incomplete
	mt := makeMapType(ikey.incomplete, ielem.incomplete, t.string())

	// TODO canonicalize t.incomplete
	t.incomplete = &mt.rtype
}

func (info iMapType) completeType(t *itype) {
	panic("unimplemented")
}
