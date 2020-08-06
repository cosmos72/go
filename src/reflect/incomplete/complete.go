// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	"unsafe"
)

// Complete completes the incomplete types in in, transforming them to a list
// of reflect.Type types. The function method is called once for each method
// added with AddMethod and should return an implementation of that method:
// a function whose first argument is the receiver.
// The list out contains the fully usable resulting types, except that methods
// can be called on them only after Complete has returned. The index indicates
// which type will be the method receiver, and stub indicates the method.
func Complete(
	in []Type,
	method func(out []reflect.Type, index int, stub Method) interface{},
) []reflect.Type {

	if method != nil {
		panic("incomplete.Complete: argument 'method' must currently be nil")
	}

	for _, t := range in {
		computeSize(t.(*itype), nil)
	}
	for i, t := range in {
		if t.(*itype).iflag&iflagSize == 0 {
			panic("incomplete.Complete: type " + strconv.Itoa(i) +
				" depends on a named type with no underlying type")
		}
	}
	for _, t := range in {
		ityp := t.(*itype)
		ityp.completeType(ityp)
	}
	return nil
}

func (u *itype) completeType(t *itype) {
	if t.complete != nil {
		return
	}
	// u.info may be another *itype with the same underlying type,
	// or one of iArrayType, iChanType ... iStructType
	u.info.completeType(t)
}

func (info iArrayType) completeType(t *itype)     {}
func (info iChanType) completeType(t *itype)      {}
func (info iInterfaceType) completeType(t *itype) {}
func (info iMapType) completeType(t *itype)       {}
func (info iFuncType) completeType(t *itype)      {}

func (info iPtrType) completeType(t *itype) {
	var iptr interface{} = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	s := t.string()
	pp.str = resolveReflectName(newName(s, "", false))
	pp.ptrToThis = 0

	t.incomplete = &pp.rtype

	ielem := info.elem.(*itype)

	t.finish = func() {
		// For the type structures linked into the binary, the
		// compiler provides a good hash of the string.
		// Create a good hash for the new string by using
		// the FNV-1 hash's mixing function to combine the
		// old hash and the new "*".
		pp.hash = fnv1(ielem.incomplete.hash, '*')
		pp.elem = ielem.incomplete
		t.complete = canonicalize(t.incomplete)
	}
}

func (info iSliceType) completeType(t *itype)  {}
func (info iStructType) completeType(t *itype) {}
