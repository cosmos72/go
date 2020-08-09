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
)

// tflag is used by an itype to signal what extra type information is available.
type iflag uint8

const (
	// iflagDefined means Define was called on the type
	iflagDefined iflag = 1 << 0

	// iflagRtype means the type has an 'incomplete' field followed in memory
	// by one of: arrayType, chanType, funcType, interfaceType, mapType, ptrType
	// sliceType, sliceType, structType as expected by reflect.
	iflagRtype = 1 << 1

	// iflagSize means the type has known size, align and fieldAlign
	iflagSize iflag = 1 << 2
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
	computeSize(t, nil)
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
		t.incomplete = &rtype{}
	}
	t.incomplete.size = size
	t.incomplete.align = align
	t.incomplete.fieldAlign = fieldAlign
	t.iflag |= iflagSize
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
	t.iflag |= iflagRtype
}

func (u *itype) completeType(t *itype) {
	if t.complete != nil {
		return
	}
	// u.info may be another *itype with the same underlying type,
	// or one of iArrayType, iChanType ... iStructType
	u.info.completeType(t)
}
