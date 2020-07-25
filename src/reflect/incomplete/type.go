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

	// unexported, marker method
	isType()
}

// analogous to reflect.Kind.
type kind uint8

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
)

// tflag is used by an itype to signal what extra type information is available.
type tflag uint8

const (
	// tflagDefined means Define was called on the type
	tflagDefined tflag = 1 << 0

	// tflagRType means the type has a known reflect.Type
	tflagRType tflag = 1 << 1

	// tflagSize means the type has known size.
	tflagSize tflag = 1 << 2
)

// itype is the implementation of Type
type itype struct {
	named   *namedType
	methods *[]Method
	size    uintptr
	kind    kind
	tflag   tflag
	// nil or one of: reflect.Type, *itype, arrayType, chanType, funcType,
	// interfaceType, mapType, ptrType, sliceType, structType
	info interface{}
}

// namedType contains the name, pkgPath and methods for named types
type namedType struct {
	name    string // name of type
	pkgPath string // import path
}

type arrayType struct {
	elem  Type
	count int
}

type chanType struct {
	elem Type
	dir  reflect.ChanDir
}

type mapType struct {
	key  Type
	elem Type
}

type ptrType struct {
	elem Type
}

type sliceType struct {
	elem Type
}

// itype methods
func (t *itype) Define(u Type) {
	if t.tflag&tflagDefined != 0 {
		panic("incomplete.Type.Define() already invoked on this type")
	}
	if t.named == nil || t.info != nil {
		panic("incomplete.Type.Define() on Type not created with NamedOf")
	}
	t.info = u.(*itype)
	descendType(t)
	computeSize(t, nil)
	t.tflag |= tflagDefined
}

func (t *itype) AddMethod(mtd Method) {
	panic("unimplemented: incomplete.Type.AddMethod()")
}

func (t *itype) isType() {
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

// ofMap is the cache for Of.
var ofMap sync.Map // map[*reflect.rtype]*itype

// Of returns a Type representing the given complete reflect.Type.
func Of(rtyp reflect.Type) Type {
	// Check the cache.
	if t, ok := ofMap.Load(rtyp); ok {
		return t.(*itype)
	}
	var named *namedType
	if rtyp.Name() != "" {
		named = &namedType{
			name:    rtyp.Name(),
			pkgPath: rtyp.PkgPath(),
		}
	}
	ityp := itype{
		named:   named,
		methods: methodsFromReflect(rtyp),
		size:    rtyp.Size(),
		kind:    kind(rtyp.Kind()),
		tflag:   tflagRType | tflagSize,
		info:    rtyp,
	}
	t, _ := ofMap.LoadOrStore(rtyp, &ityp)
	return t.(*itype)
}

// NamedOf creates the incomplete type with the specified name and package path.
// The name can be bound to an underlying type with the Define method.
func NamedOf(name, pkgPath string) Type {
	return &itype{
		named: &namedType{
			name:    name,
			pkgPath: pkgPath,
		},
		methods: nil,
		size:    0,
		kind:    kInvalid,
		tflag:   tflag(0),
		info:    nil,
	}
}

// ArrayOf creates an incomplete array type with the given count and
// element type described by elem.
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("incomplete.ArrayOf: element count is negative")
	}
	ielem := elem.(*itype)
	if ielem.tflag&tflagRType != 0 {
		return Of(reflect.ArrayOf(
			count,
			ielem.info.(reflect.Type),
		))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    uintptr(count) * ielem.size,
		kind:    kArray,
		tflag:   ielem.tflag & tflagSize,
		info: arrayType{
			elem:  elem,
			count: count,
		},
	}
}

const sizeOfChan = unsafe.Sizeof(make(chan int))

// ChanOf is analogous to reflect.ChanOf.
func ChanOf(dir reflect.ChanDir, elem Type) Type {
	ielem := elem.(*itype)
	if ielem.tflag&tflagRType != 0 {
		return Of(reflect.ChanOf(
			dir,
			ielem.info.(reflect.Type),
		))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfChan,
		kind:    kChan,
		tflag:   tflagSize,
		info: chanType{
			elem: elem,
			dir:  dir,
		},
	}
}

const sizeOfMap = unsafe.Sizeof(make(map[int]int))

// MapOf creates an incomplete map type with the given key and element types.
func MapOf(key, elem Type) Type {
	ikey := key.(*itype)
	ielem := elem.(*itype)
	if ikey.tflag&ielem.tflag&tflagRType != 0 {
		return Of(reflect.MapOf(
			ikey.info.(reflect.Type),
			ielem.info.(reflect.Type),
		))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfMap,
		kind:    kMap,
		tflag:   tflagSize,
		info: mapType{
			key:  key,
			elem: elem,
		},
	}
}

const sizeOfPtr = unsafe.Sizeof(new(int))

// PtrTo is analogous to reflect.PtrTo.
func PtrTo(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.tflag&tflagRType != 0 {
		return Of(reflect.PtrTo(
			ielem.info.(reflect.Type),
		))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfPtr,
		kind:    kPtr,
		tflag:   tflagSize,
		info: ptrType{
			elem: elem,
		},
	}
}

const sizeOfSlice = unsafe.Sizeof(make([]int, 0))

// SliceOf is analogous to reflect.SliceOf.
func SliceOf(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.tflag&tflagRType != 0 {
		return Of(reflect.SliceOf(
			ielem.info.(reflect.Type),
		))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfSlice,
		kind:    kSlice,
		tflag:   tflagSize,
		info: sliceType{
			elem: elem,
		},
	}
}
