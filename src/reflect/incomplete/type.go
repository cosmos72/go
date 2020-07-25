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

// A kind represents the specific kind of type that a Type represents.
// The zero kind is not a valid kind.
type kind uint8

const (
	kInvalid kind = iota
	kBool
	kInt
	kInt8
	kInt16
	kInt32
	kInt64
	kUint
	kUint8
	kUint16
	kUint32
	kUint64
	kUintptr
	kFloat32
	kFloat64
	kComplex64
	kComplex128
	kArray
	kChan
	kFunc
	kInterface
	kMap
	kPtr
	kSlice
	kString
	kStruct
	kUnsafePointer
)

// tflag is used by an itype to signal what extra type information is available.
type tflag uint8

const (
	// tflagSize means the type has known size.
	tflagSize tflag = 1 << 0

	// tflagDefined means Define was called on the type
	tflagDefined tflag = 1 << 1
)

// itype is the implementation of Type
type itype struct {
	named   *namedType
	methods *[]Method
	size    uintptr
	kind    kind
	tflag   tflag
	// one of: reflect.Type, arrayType, chanType, funcType,
	// interfaceType, mapType, ptrType, sliceType, structType
	extra interface{}
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

type funcType struct {
	in       []Type
	out      []Type
	variadic bool
}

type interfaceType struct {
	embedded []Type
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

type structType struct {
	fields []StructField
}

// itype methods
func (t *itype) Define(u Type) {
	panic("unimplemented: incomplete.Type.Define()")
}

func (t *itype) AddMethod(mtd Method) {
	panic("unimplemented: incomplete.Type.AddMethod()")
}

func (t *itype) isType() {
}

// Of returns a Type representing the given complete reflect.Type.
func Of(rtyp reflect.Type) Type {
	var named *namedType
	if rtyp.Name() != "" {
		named = &namedType{
			name:    rtyp.Name(),
			pkgPath: rtyp.PkgPath(),
		}
	}
	return &itype{
		named:   named,
		methods: methodsFromReflect(rtyp),
		size:    rtyp.Size(),
		tflag:   tflagSize | tflagDefined,
		extra:   rtyp,
	}
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
		tflag:   tflag(0),
		extra:   nil,
	}
}

// ArrayOf creates an incomplete array type with the given count and
// element type described by elem.
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("incomplete.ArrayOf: element count is negative")
	}

	ielem := elem.(*itype)
	return &itype{
		named:   nil,
		methods: nil,
		size:    uintptr(count) * ielem.size,
		tflag:   ielem.tflag & tflagSize,
		extra: arrayType{
			elem:  elem,
			count: count,
		},
	}
}

const sizeOfChan = unsafe.Sizeof(make(chan int))

// ChanOf is analogous to reflect.ChanOf.
func ChanOf(dir reflect.ChanDir, elem Type) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfChan,
		tflag:   tflagSize,
		extra: chanType{
			elem: elem,
			dir:  dir,
		},
	}
}

const sizeOfFunc = unsafe.Sizeof(func() {})

// FuncOf is analogous to reflect.FuncOf.
func FuncOf(in, out []Type, variadic bool) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfFunc,
		tflag:   tflagSize,
		extra: funcType{
			// safety: make a copy of in[] and out[]
			in:       append([]Type{}, in...),
			out:      append([]Type{}, out...),
			variadic: variadic,
		},
	}
}

// InterfaceOf returns an incomplete interface type with the given list of
// named interface types. InterfaceOf panics if one of the given embedded types
// is unnamed or its kind is not reflect.Interface. It also panics if types
// with distinct, non-empty package paths are embedded.
//
// Explicit methods can be added with AddMethod.
func InterfaceOf(embedded []Type) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    0, // size of interfaces can vary?
		tflag:   tflag(0),
		extra: interfaceType{
			// safety: make a copy of embedded[]
			embedded: append([]Type{}, embedded...),
		},
	}
}

const sizeOfMap = unsafe.Sizeof(make(map[int]int))

// MapOf creates an incomplete map type with the given key and element types.
func MapOf(key, elem Type) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfMap,
		tflag:   tflagSize,
		extra: mapType{
			key:  key,
			elem: elem,
		},
	}
}

const sizeOfPtr = unsafe.Sizeof(new(int))

// PtrTo is analogous to reflect.PtrTo.
func PtrTo(t Type) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfPtr,
		tflag:   tflagSize,
		extra: ptrType{
			elem: t,
		},
	}
}

const sizeOfSlice = unsafe.Sizeof(make([]int, 0))

// SliceOf is analogous to reflect.SliceOf.
func SliceOf(t Type) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfSlice,
		tflag:   tflagSize,
		extra: sliceType{
			elem: t,
		},
	}
}

// StructOf is analogous to reflect.StructOf.
func StructOf(fields []StructField) Type {
	return &itype{
		named:   nil,
		methods: nil,
		size:    0,
		tflag:   tflag(0),
		extra: structType{
			// safety: make a copy of fields[]
			fields: append([]StructField{}, fields...),
		},
	}
}
