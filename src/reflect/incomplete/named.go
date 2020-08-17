// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"unsafe"
)

const maxMethods = 128

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

type basicTypeUncommon struct {
	rtype
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type arrayTypeUncommon struct {
	arrayType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type chanTypeUncommon struct {
	chanType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type funcTypeUncommon struct {
	funcType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type mapTypeUncommon struct {
	mapType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type ptrTypeUncommon struct {
	ptrType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type sliceTypeUncommon struct {
	sliceType
	uncommon uncommonType
	method   [maxMethods]*rtype
}

type structTypeUncommon struct {
	structType
	uncommon uncommonType
	method   [maxMethods]*rtype
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

func (n *namedType) String() string {
	return n.str
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
	allocUncommonType(t)
	t.computeSize(t, nil) // early check for forbidden loops
	t.iflag |= iflagDefined
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
		if t1 != nil && t1 == t2 {
			t.info = nil
			panic("incomplete.Type.Define(): invalid Type loop")
		}
	}
	t.info = last
}

func allocUncommonType(t *itype) {
	u := t.info.(*itype)
	rtu := u.incomplete
	if rtu == nil {
		if u.complete == nil {
			panic("internal error: fields .complete and .incomplete are both nil in type")
		}
		rtu = unwrap(u.complete)
	}
	var uncommon *uncommonType
	var rt *rtype
	switch kind := u.kind(); kind {
	case kInvalid:
		panic("incomplete.Type.Define: underlying type is not Define'd yet")
	case kArray:
		array := arrayTypeUncommon{
			arrayType: *(*arrayType)(unsafe.Pointer(rtu)),
		}
		uncommon = &array.uncommon
		rt = &array.rtype
	case kChan:
		ch := chanTypeUncommon{
			chanType: *(*chanType)(unsafe.Pointer(rtu)),
		}
		uncommon = &ch.uncommon
		rt = &ch.rtype
	case kFunc:
		fn := funcTypeUncommon{
			funcType: *(*funcType)(unsafe.Pointer(rtu)),
		}
		uncommon = &fn.uncommon
		rt = &fn.rtype
	case kInterface:
		panic("unimplemented: named interface type")
	case kMap:
		m := mapTypeUncommon{
			mapType: *(*mapType)(unsafe.Pointer(rtu)),
		}
		uncommon = &m.uncommon
		rt = &m.rtype
	case kPtr:
		ptr := ptrTypeUncommon{
			ptrType: *(*ptrType)(unsafe.Pointer(rtu)),
		}
		uncommon = &ptr.uncommon
		rt = &ptr.rtype
	case kSlice:
		slice := sliceTypeUncommon{
			sliceType: *(*sliceType)(unsafe.Pointer(rtu)),
		}
		uncommon = &slice.uncommon
		rt = &slice.rtype
	case kStruct:
		st := structTypeUncommon{
			structType: *(*structType)(unsafe.Pointer(rtu)),
		}
		uncommon = &st.uncommon
		rt = &st.rtype
	default:
		/*
			println("allocUncommonType for kind = " +
				reflect.Kind(rtu.kind&kindMask).String() + " underlying = " +
				u.string())
		*/
		bt := basicTypeUncommon{
			rtype: *rtu,
		}
		uncommon = &bt.uncommon
		rt = &bt.rtype
	}
	uncommon.moff = uint32(unsafe.Sizeof(uncommonType{}))
	rt.hash = 0
	rt.tflag = (rt.tflag | tflagUncommon | tflagNamed) & ^tflagExtraStar
	rt.str = 0
	rt.ptrToThis = 0
	t.incomplete = rt
}

func (t *uncommonType) methods() []method {
	if t.mcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
}

func (t *uncommonType) exportedMethods() []method {
	if t.xcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.xcount > 0"))[:t.xcount:t.xcount]
}

// add returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}
