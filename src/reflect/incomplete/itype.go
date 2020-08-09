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
	prepareRtype(*itype)
	completeType(*itype)
}

// itype methods
func (t *itype) Define(u Type) {
	if t.iflag&iflagDefined != 0 {
		panic("incomplete.Type.Define: already invoked on this type")
	}
	if t.named == nil {
		panic("incomplete.Type.Define: type not created with NamedOf")
	}
	if t.complete != nil {
		panic("incomplete.Type.Define: type is already complete")
	}
	t.info = u.(*itype)
	descendType(t)
	t.computeSize(t, nil)
	t.iflag |= iflagDefined
}

func (t *itype) AddMethod(mtd Method) {
	if t.named == nil {
		panic("incomplete.Type.AddMethod: type not created with NamedOf")
	}
	if t.complete != nil {
		panic("incomplete.Type.AddMethod: type is already complete")
	}
	t.named.vmethod = append(t.named.vmethod, mtd)
}

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
		// FIXME allocate the correct *Type
		t.incomplete = &rtype{}
	}
	t.incomplete.size = size
	t.incomplete.align = align
	t.incomplete.fieldAlign = fieldAlign
	t.iflag |= iflagSize
}

func makeQname(name, pkgPath string) qname {
	str := name
	if pkgPath != "" {
		str = pkgPath + "." + name
		// slightly reduce memory usage
		pkgPath = str[:len(pkgPath)]
		name = str[1+len(pkgPath):]
		str = filename(str)
	}
	return qname{
		name:    name,
		pkgPath: pkgPath,
		str:     str,
	}
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
	if t.iflag&iflagSize != 0 {
		return true
	}
	if u.info == nil {
		return false
	}
	push(t, work)
	ok := u.info.computeSize(t, work)
	delete(work, t)
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

// prepareRtype replaces t.incomplete with an *rtype followed in memory
// by one of: arrayType, chanType, funcType, interfaceType, mapType, ptrType
// sliceType, sliceType, structType as expected by reflect.
//
// it also sets t.incomplete.hash
func (u *itype) prepareRtype(t *itype) {
	if t.complete != nil || t.iflag&iflagRtype != 0 {
		return
	}
	// u.info may be another *itype with the same underlying type as t,
	// or one of iArrayType, iChanType ... iStructType
	u.info.prepareRtype(t)
}

func (u *itype) completeType(t *itype) {
	if t.complete != nil {
		return
	}
	// u.info may be another *itype with the same underlying type,
	// or one of iArrayType, iChanType ... iStructType
	u.info.completeType(t)
}
