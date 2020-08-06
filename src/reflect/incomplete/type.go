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

	// unexported
	kind() kind

	// unexported
	string() string

	// unexported
	printable
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

	// iflagSize means the type has known size, align and fieldAlign
	iflagSize iflag = 1 << 1
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
	// nil or one of: *itype, iArrayType, iChanType, iFuncType,
	// iInterfaceType, iMapType, iPtrType, iSliceType, iStructType
	info printable
}

// namedType contains the name, pkgPath and methods for named types
type namedType struct {
	name    string   // name of type
	pkgPath string   // import path
	str     string   // string representation
	method  []Method // methods
}

type iArrayType struct {
	elem  Type
	count int
}

type iChanType struct {
	elem Type
	dir  reflect.ChanDir
}

type iMapType struct {
	key  Type
	elem Type
}

type iPtrType struct {
	elem Type
}

type iSliceType struct {
	elem Type
}

// itype methods
func (t *itype) Define(u Type) {
	if t.iflag&iflagDefined != 0 {
		panic("incomplete.Type.Define() already invoked on this type")
	}
	if t.named == nil || t.info != nil {
		panic("incomplete.Type.Define() on Type not created with NamedOf")
	}
	t.info = u.(*itype)
	descendType(t)
	computeSize(t, nil)
	t.iflag |= iflagDefined
}

func (t *itype) AddMethod(mtd Method) {
	panic("unimplemented: incomplete.Type.AddMethod()")
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
			name:    rtyp.Name(),
			pkgPath: rtyp.PkgPath(),
			str:     rtyp.String(),
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
		named.method = methodsFromReflect(rtyp)
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
	str := name
	if pkgPath != "" {
		str = pkgPath + "." + name
		// slightly reduce memory usage
		pkgPath = str[:len(pkgPath)]
		name = str[1+len(pkgPath):]
		str = filename(str)
	}
	return &itype{
		named: &namedType{
			name:    name,
			pkgPath: pkgPath,
			str:     str,
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

// ArrayOf creates an incomplete array type with the given count and
// element type described by elem.
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("incomplete.ArrayOf: element count is negative")
	}
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ArrayOf(count, ielem.complete))
	}
	return &itype{
		named:      nil,
		comparable: ielem.comparable,
		iflag:      ielem.iflag & iflagSize,
		incomplete: &rtype{
			size: uintptr(count) * ielem.size(),
			kind: kArray,
		},
		info: iArrayType{
			elem:  elem,
			count: count,
		},
	}
}

var rtypeChan *rtype = unwrap(reflect.TypeOf(make(chan unsafe.Pointer)))

// ChanOf is analogous to reflect.ChanOf.
func ChanOf(dir reflect.ChanDir, elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ChanOf(dir, ielem.complete))
	}
	incomplete := *rtypeChan
	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &incomplete,
		info: iChanType{
			elem: elem,
			dir:  dir,
		},
	}
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
		panic("incomplete.MapOf: invalid key type, cannot be compared with itself")
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

var rtypePtr *rtype = unwrap(reflect.TypeOf(new(unsafe.Pointer)))

// PtrTo is analogous to reflect.PtrTo.
func PtrTo(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.PtrTo(ielem.complete))
	}
	incomplete := *rtypePtr
	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &incomplete,
		info: iPtrType{
			elem: elem,
		},
	}
}

var rtypeSlice *rtype = unwrap(reflect.TypeOf(make([]unsafe.Pointer, 0)))

// SliceOf is analogous to reflect.SliceOf.
func SliceOf(elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.SliceOf(ielem.complete))
	}
	incomplete := *rtypeSlice
	return &itype{
		named:      nil,
		incomplete: &incomplete,
		comparable: tfalse,
		iflag:      iflagSize,
		info: iSliceType{
			elem: elem,
		},
	}
}
