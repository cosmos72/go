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
)

// Type represents an incomplete type, or part of an incomplete composite type.
// It is a safe way to define the layout of (possibly recursive) types
// with the Of, NamedOf, ArrayOf, ChanOf, FuncOf, InterfaceOf, MapOf, PtrTo,
// SliceOf, and StructOf functions before the actual types are created with
// Complete.
type Type interface {
	// Define sets the underlying type of an incomplete named type
	// to the underlying type of the argument 'u'. It panics if the receiver
	// is not a named type, or Define was already called on the receiver,
	// or if the result would contain an invalid recursion.
	Define(u Type)

	// AddMethod adds the given method to this type. The Index field of the given
	// method is ignored. It panics if there is a method name clash, or if
	// methods with distinct, non-empty PkgPath strings are added. Furthermore,
	// one of the following cases must apply:
	//
	// Case 1: this type was created with InterfaceOf.
	//
	// Case 2: this type was created with NamedOf and defined to a non-pointer,
	// non-interface type.
	//
	// Case 3: this type was created with PtrTo, with an element type which
	// Case 2 applies to.
	AddMethod(mtd Method)

	// unexported
	kind() kind

	// unexported
	string() string

	// unexported
	iAnyType
}

// analogous to reflect.Kind.
type kind = uint8

const (
	kInvalid       = kind(reflect.Invalid)
	kBool          = kind(reflect.Bool)
	kInt           = kind(reflect.Int)
	kInt8          = kind(reflect.Int8)
	kInt16         = kind(reflect.Int16)
	kInt32         = kind(reflect.Int32)
	kInt64         = kind(reflect.Int64)
	kUint          = kind(reflect.Uint)
	kUint8         = kind(reflect.Uint8)
	kUint16        = kind(reflect.Uint16)
	kUint32        = kind(reflect.Uint32)
	kUint64        = kind(reflect.Uint64)
	kUintptr       = kind(reflect.Uintptr)
	kFloat32       = kind(reflect.Float32)
	kFloat64       = kind(reflect.Float64)
	kComplex64     = kind(reflect.Complex64)
	kComplex128    = kind(reflect.Complex128)
	kArray         = kind(reflect.Array)
	kChan          = kind(reflect.Chan)
	kFunc          = kind(reflect.Func)
	kInterface     = kind(reflect.Interface)
	kMap           = kind(reflect.Map)
	kPtr           = kind(reflect.Ptr)
	kSlice         = kind(reflect.Slice)
	kString        = kind(reflect.String)
	kStruct        = kind(reflect.Struct)
	kUnsafePointer = kind(reflect.UnsafePointer)

	kindDirectIface = 1 << 5
	kindGCProg      = 1 << 6 // Type.gc points to GC program
	kindMask        = (1 << 5) - 1
)

// tflag is used by an itype to signal what extra type information is available.
type iflag uint8

const (
	// iflagDefined means Define was called on the type
	iflagDefined iflag = 1 << 0

	// iflagSize means the type has known fields: size, align, fieldAlign
	// and ptrdata
	iflagSize iflag = 1 << 1

	// iflagHashStr means the type has known fields: hash and str.
	iflagHashStr = 1 << 2
)

// tribool is a three-valued boolean: true, false, unknown
type tribool uint8

const (
	tunknown tribool = 0
	tfalse   tribool = 1
	ttrue    tribool = 2
)

func makeTribool(flag bool) tribool {
	if flag {
		return ttrue
	} else {
		return tfalse
	}
}

func andTribool(a tribool, b tribool) tribool {
	if a == tunknown || b == tunknown {
		return tunknown
	} else if a == tfalse || b == tfalse {
		return tfalse
	} else {
		return ttrue
	}
}

func (flag tribool) String() string {
	switch flag {
	case tfalse:
		return "tfalse"
	case ttrue:
		return "ttrue"
	}
	return "tunknown"
}

// ofMap is the cache for Of.
var ofMap = map[reflect.Type]*itype{}
var ofMutex sync.Mutex

// Of returns a Type representing the given complete reflect.Type.
func Of(rtyp reflect.Type) Type {
	ofMutex.Lock()
	defer ofMutex.Unlock()
	return of(rtyp)
}

func of(rtyp reflect.Type) Type {
	// Check the cache.
	if t, ok := ofMap[rtyp]; ok {
		return t
	}
	var named *namedType
	if rtyp.Name() != "" {
		named = &namedType{
			qname: qname{
				name:    rtyp.Name(),
				pkgPath: rtyp.PkgPath(),
				str:     rtyp.String(),
			},
		}
	}
	ityp := &itype{
		named:      named,
		comparable: makeTribool(rtyp.Comparable()),
		iflag:      iflagSize,
		complete:   rtyp,
		info:       nil,
	}
	ofMap[rtyp] = ityp
	if named != nil {
		// convert methods after updating cache - avoids infinite recursion
		named.vmethod = methodsFromReflect(rtyp)
		named.pmethod = methodsFromReflect(reflect.PtrTo(rtyp))
	}
	return ityp
}

// NamedOf creates a new incomplete type with the specified name and package path.
// The returned type can be bound to an underlying type calling its Define method.
func NamedOf(name, pkgPath string) Type {
	if name == "" {
		panic("incomplete.NamedOf: empty name")
	}
	if !isValidFieldName(name) {
		panic("incomplete.NamedOf: invalid name")
	}
	return &itype{
		named: &namedType{
			qname: makeQname(name, pkgPath),
		},
	}
}

// filename returns the trailing portion of path after the last '/'
func filename(path string) string {
	n := len(path)
	for i := n - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func descendType(t *itype) {
	next := func(ityp *itype) *itype {
		var ret *itype
		if ityp != nil {
			ret, _ = ityp.info.(*itype)
		}
		return ret
	}
	t1, t2, last := t, t, t
	for t1 != nil {
		last = t1
		t1 = next(t1)
		t2 = next(next(t2))
		if t1 == t2 {
			t.info = nil
			panic("incomplete.Type.Define(): invalid Type loop")
		}
	}
	t.info = last
}

func computeSize(t *itype, work map[*itype]struct{}) bool {
	if t.iflag&iflagSize != 0 {
		return true
	}
	return t.computeSize(t, work)
}

func computeHashStr(t *itype) {
	if t.complete != nil || t.iflag&iflagHashStr != 0 {
		return
	}
	t.computeHashStr(t)
	t.iflag |= iflagHashStr
}

func completeType(t *itype) {
	if t.complete != nil {
		return
	}
	t.completeType(t)
}
