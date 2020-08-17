// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflect/incomplete implements run-time creation of recursive types,
// which is not possible with reflect API alone.
//
// See "The Laws of Reflection" for an introduction to reflection in Go:
// https://golang.org/doc/articles/laws_of_reflection.html
package incomplete

import (
	"reflect"
	"sync"
	"unsafe"
)

// itype is the implementation of Type
type itype struct {
	named      *namedType
	comparable tribool
	iflag      iflag
	incomplete *rtype
	complete   reflect.Type // nil if not known yet
	info       iAnyType     // always non-nil
}

// namedType contains the name, pkgPath and methods for named types
type namedType struct {
	qname            // name of type and import path
	vmethod []Method // methods with value receiver
	pmethod []Method // methods with pointer receiver
}

// qname is a qualified name, i.e. pkgPath and name
type qname struct {
	name    string
	pkgPath string
	str     string // string representation
}

// one of: *itype, iArrayType, iChanType, iFuncType,
// iInterfaceType, iMapType, iPtrType, iSliceType, iStructType
type iAnyType interface {
	printTo(dst []byte, separator string) []byte
	computeSize(t *itype, work map[*itype]struct{}) bool
	computeHashStr(*itype)
	completeType(*itype)
}

// The lookupCache caches ArrayOf, ChanOf, MapOf, PtrTo and SliceOf calls
// and canonicalizes their return values
var lookupCache sync.Map // map[cacheKey]*itype

// A cacheKey is the key for use in the lookupCache.
// Four values describe any of the types we are looking for:
// type kind, one or two subtypes, and an extra integer.
type cacheKey struct {
	kind  kind
	t1    *itype
	t2    *itype
	extra uintptr
}

func canonicalize(ckey cacheKey, t *itype) Type {
	ret, _ := lookupCache.LoadOrStore(ckey, t)
	return ret.(Type)
}

// itype methods
func (t *itype) kind() kind {
	if t.complete != nil {
		return kind(t.complete.Kind())
	} else if t.incomplete != nil {
		return t.incomplete.kind
	} else {
		return kInvalid
	}
}

func (t *itype) string() string {
	return string(t.printTo(([]byte)(nil), ""))
}

func (t *itype) size() uintptr {
	if t.iflag&iflagSize == 0 {
		return 0 // not known yet
	} else if t.complete != nil {
		return t.complete.Size()
	} else if t.incomplete != nil {
		return t.incomplete.size
	} else {
		panic("reflect/incomplete error: Type size should be known, but it is not")
	}
}

func (t *itype) align() uint8 {
	if t.iflag&iflagSize == 0 {
		return 0 // not known yet
	} else if t.complete != nil {
		return uint8(t.complete.Align())
	} else if t.incomplete != nil {
		return t.incomplete.align
	} else {
		panic("reflect/incomplete error: Type align should be known, but it is not")
	}
}

func (t *itype) fieldAlign() uint8 {
	if t.iflag&iflagSize == 0 {
		return 0 // not known yet
	} else if t.complete != nil {
		return uint8(t.complete.FieldAlign())
	} else if t.incomplete != nil {
		return t.incomplete.fieldAlign
	} else {
		panic("reflect/incomplete error: Type fieldAlign should be known, but it is not")
	}
}

func (t *itype) setSize(size uintptr, align uint8, fieldAlign uint8) {
	if t.incomplete == nil {
		panic("itype.incomplete is nil")
	}
	t.incomplete.size = size
	t.incomplete.align = align
	t.incomplete.fieldAlign = fieldAlign
	t.iflag |= iflagSize
}

func (t *itype) setSizeFrom(rfrom *rtype) bool {
	t.setSize(rfrom.size, rfrom.align, rfrom.fieldAlign)
	return true
}

func (t *itype) setHashStrFromNamed(named *namedType) {
	if t.incomplete == nil {
		panic("itype.incomplete is nil")
	}
	hash := uintptr(unsafe.Pointer(t.incomplete))
	t.incomplete.hash = fnv4(uint32(hash>>32), uint32(hash))
	t.incomplete.str = resolveReflectName(newName(named.str, "", false))
	// println("setHashStrFromNamed: " + named.str + " -> " + wrap(t.incomplete).String())
}

func (t *itype) printTo(bytes []byte, separator string) []byte {
	bytes = append(bytes, separator...)
	if t.complete != nil {
		return append(bytes, t.complete.String()...)
	} else if t.named != nil {
		return append(bytes, t.named.str...)
	} else if t.info != nil {
		return t.info.printTo(bytes, "")
	} else {
		panic("reflect/incomplete error: Type string representation should be known, but it is not")
	}
}

func (u *itype) computeSize(t *itype, work map[*itype]struct{}) bool {
	if t.complete != nil || t.iflag&iflagSize != 0 {
		return true
	} else if u.complete != nil {
		return t.setSizeFrom(unwrap(u.complete))
	} else if u.incomplete != nil && u.iflag&iflagSize != 0 {
		return t.setSizeFrom(u.incomplete)
	}
	if t == u {
		work = push(t, work)
	}
	// forward the call to u.info
	ok := u.info.computeSize(t, work)
	if t == u {
		delete(work, t)
	}
	return ok
}

func push(t *itype, work map[*itype]struct{}) map[*itype]struct{} {
	if work == nil {
		work = make(map[*itype]struct{})
	} else if _, ok := work[t]; ok {
		panic("invalid Type loop detected: cannot compute size")
	}
	work[t] = struct{}{}
	return work
}

// computeHashStr fills t.incomplete fields 'hash' and 'str'
func (u *itype) computeHashStr(t *itype) {
	if t.complete != nil || t.iflag&iflagHashStr != 0 {
		return
	} else if t.named != nil {
		t.setHashStrFromNamed(t.named)
		return
	}
	// u.info may be another *itype with the same underlying type as t,
	// or one of iArrayType, iChanType ... iStructType

	u.info.computeHashStr(t)

	t.iflag |= iflagHashStr
}

func (u *itype) completeType(t *itype) {
	if t.complete != nil {
		// nothing to do
	} else if t.named != nil {
		t.complete = wrap(t.incomplete)
		completeMethods(t)
	} else {
		// u.info may be another *itype with the same underlying type,
		// or one of iArrayType, iChanType ... iStructType
		u.info.completeType(t)
	}
}
