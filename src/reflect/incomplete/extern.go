// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

// Make sure these routines stay in sync with ../../../runtime/map.go!
// These types exist only for GC, so we only fill out GC relevant info.
// Currently, that's just size and the GC program. We also fill in string
// for possible debugging use.
const (
	maxKeySize uintptr = 128
	maxValSize uintptr = 128
)

const ptrSize = 4 << (^uintptr(0) >> 63) // unsafe.Sizeof(uintptr(0)) but an ideal const

//go:linkname bucketOf reflect.bucketOf
func bucketOf(ktyp, etyp *rtype) *rtype

// convert *incomplete.rtype to reflect.Type and canonicalize it
//go:linkname canonicalize reflect.toType
func canonicalize(t *rtype) reflect.Type

//go:linkname fnv1 reflect.fnv1
func fnv1(x uint32, list ...byte) uint32

//go:linkname hashMightPanic reflect.hashMightPanic
func hashMightPanic(t *rtype) bool

//go:linkname needKeyUpdate reflect.needKeyUpdate
func needKeyUpdate(t *rtype) bool

//go:linkname newName reflect.newName
func newName(n, tag string, exported bool) name

//go:linkname isReflexive reflect.isReflexive
func isReflexive(t *rtype) bool

//go:linkname isValidFieldName reflect.isValidFieldName
func isValidFieldName(fieldName string) bool

//go:linkname resolveReflectName reflect.resolveReflectName
func resolveReflectName(n name) nameOff

//go:noescape
//go:linkname typehash reflect.typehash
func typehash(t *rtype, p unsafe.Pointer, h uintptr) uintptr

// convert reflect.Type to *incomplete.rtype
func unwrap(t reflect.Type) *rtype {
	return *(**rtype)(unsafe.Pointer(&t))
}
