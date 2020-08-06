// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"unsafe"
)

type nameOff int32 // offset to a name
type typeOff int32 // offset to an *rtype
type textOff int32 // offset from top of text section

// tflag is used by an rtype to signal what extra type information is
// available in the memory directly following the rtype value.
//
// tflag values must be kept in sync with copies in:
//	cmd/compile/internal/gc/reflect.go
//	cmd/link/internal/ld/decodesym.go
//	runtime/type.go
type tflag uint8

const (
	// tflagUncommon means that there is a pointer, *uncommonType,
	// just beyond the outer type structure.
	//
	// For example, if t.Kind() == Struct and t.tflag&tflagUncommon != 0,
	// then t has uncommonType data and it can be accessed as:
	//
	//	type tUncommon struct {
	//		structType
	//		u uncommonType
	//	}
	//	u := &(*tUncommon)(unsafe.Pointer(t)).u
	tflagUncommon tflag = 1 << 0

	// tflagExtraStar means the name in the str field has an
	// extraneous '*' prefix. This is because for most types T in
	// a program, the type *T also exists and reusing the str data
	// saves binary size.
	tflagExtraStar tflag = 1 << 1

	// tflagNamed means the type has a name.
	tflagNamed tflag = 1 << 2

	// tflagRegularMemory means that equal and hash functions can treat
	// this type as a single region of t.size bytes.
	tflagRegularMemory tflag = 1 << 3
)

// rtype is the common implementation of most values.
// It is embedded in other struct types.
//
// rtype must be kept in sync with ../../runtime/type.go:/^type._type.
type rtype struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      tflag   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte   // garbage collection data
	str       nameOff // string form
	ptrToThis typeOff // type for pointer to this type, may be zero
}

// Method on non-interface type
type method struct {
	name nameOff // name of method
	mtyp typeOff // method type (without receiver)
	ifn  textOff // fn used in interface call (one-word receiver)
	tfn  textOff // fn used for normal method call
}

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type uncommonType struct {
	pkgPath nameOff // import path; empty for built-in types like int, string
	mcount  uint16  // number of methods
	xcount  uint16  // number of exported methods
	moff    uint32  // offset from this uncommontype to [mcount]method
	_       uint32  // unused
}

// arrayType represents a fixed array type.
type arrayType struct {
	rtype
	elem  *rtype // array element type
	slice *rtype // slice type
	len   uintptr
}

// chanType represents a channel type.
type chanType struct {
	rtype
	elem *rtype  // channel element type
	dir  uintptr // channel direction (reflect.ChanDir)
}

// funcType represents a function type.
//
// A *rtype for each in and out parameter is stored in an array that
// directly follows the funcType (and possibly its uncommonType). So
// a function type with one method, one input, and one output is:
//
//	struct {
//		funcType
//		uncommonType
//		[2]*rtype    // [0] is in, [1] is out
//	}
type funcType struct {
	rtype
	inCount  uint16
	outCount uint16 // top bit is set if last input parameter is ...
}

// imethod represents a method on an interface type
type imethod struct {
	name nameOff // name of method
	typ  typeOff // .(*FuncType) underneath
}

// interfaceType represents an interface type.
type interfaceType struct {
	rtype
	pkgPath name      // import path
	methods []imethod // sorted by hash
}

// mapType represents a map type.
type mapType struct {
	rtype
	key    *rtype // map key type
	elem   *rtype // map element (value) type
	bucket *rtype // internal bucket structure
	// function for hashing keys (ptr to key, seed) -> hash
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8  // size of key slot
	valuesize  uint8  // size of value slot
	bucketsize uint16 // size of bucket
	flags      uint32
}

// ptrType represents a pointer type.
type ptrType struct {
	rtype
	elem *rtype // pointer element (pointed at) type
}

// sliceType represents a slice type.
type sliceType struct {
	rtype
	elem *rtype // slice element type
}

// Struct field
type structField struct {
	name        name    // name is always non-empty
	typ         *rtype  // type of field
	offsetEmbed uintptr // byte offset of field<<1 | isEmbedded
}

func (f *structField) offset() uintptr {
	return f.offsetEmbed >> 1
}

func (f *structField) embedded() bool {
	return f.offsetEmbed&1 != 0
}

// structType represents a struct type.
type structType struct {
	rtype
	pkgPath name
	fields  []structField // sorted by offset
}

// name is an encoded type name with optional extra data.
//
// The first byte is a bit field containing:
//
//	1<<0 the name is exported
//	1<<1 tag data follows the name
//	1<<2 pkgPath nameOff follows the name and tag
//
// The next two bytes are the data length:
//
//	 l := uint16(data[1])<<8 | uint16(data[2])
//
// Bytes [3:3+l] are the string data.
//
// If tag data follows then bytes 3+l and 3+l+1 are the tag length,
// with the data following.
//
// If the import path follows, then 4 bytes at the end of
// the data form a nameOff. The import path is only set for concrete
// methods that are defined in a different package than their type.
//
// If a name starts with "*", then the exported bit represents
// whether the pointed to type is exported.
type name struct {
	bytes *byte
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

func (info iArrayType) prepareRtype(t *itype) {
	panic("unimplemented")
}

func (info iChanType) prepareRtype(t *itype) {
	ielem := info.elem.(*itype)
	ielem.prepareRtype(ielem)

	// Make a channel type.
	var ichan interface{} = (chan unsafe.Pointer)(nil)
	prototype := *(**chanType)(unsafe.Pointer(&ichan))
	ch := *prototype
	ch.tflag = tflagRegularMemory
	ch.dir = uintptr(info.dir)
	s := t.string()
	ch.str = resolveReflectName(newName(s, "", false))
	ch.hash = fnv1(ielem.incomplete.hash, 'c', byte(info.dir))

	// TODO canonicalize ielem.incomplete and t.incomplete
	ch.elem = ielem.incomplete
	t.incomplete = &ch.rtype
}

func (info iInterfaceType) prepareRtype(t *itype) {
	panic("unimplemented")
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

func (info iFuncType) prepareRtype(t *itype) {
	panic("unimplemented")
}

func (info iPtrType) prepareRtype(t *itype) {
	ielem := info.elem.(*itype)
	ielem.prepareRtype(ielem)

	var iptr interface{} = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	s := t.string()
	pp.str = resolveReflectName(newName(s, "", false))
	pp.ptrToThis = 0

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	pp.hash = fnv1(ielem.incomplete.hash, '*')

	// TODO canonicalize ielem.incomplete and t.incomplete
	pp.elem = ielem.incomplete
	t.incomplete = &pp.rtype
}

func (info iSliceType) prepareRtype(t *itype) {
	ielem := info.elem.(*itype)
	ielem.prepareRtype(ielem)

	// Make a slice type.
	var islice interface{} = ([]unsafe.Pointer)(nil)
	prototype := *(**sliceType)(unsafe.Pointer(&islice))
	slice := *prototype
	slice.tflag = 0
	s := t.string()
	slice.str = resolveReflectName(newName(s, "", false))
	slice.hash = fnv1(ielem.incomplete.hash, '[')
	slice.ptrToThis = 0

	// TODO canonicalize ielem.incomplete and t.incomplete
	slice.elem = ielem.incomplete
	t.incomplete = &slice.rtype
}

func (info iStructType) prepareRtype(t *itype) {
	panic("unimplemented")
}
