// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type iFuncType struct {
	in       []Type
	out      []Type
	rargs    []*rtype // slice where in+out reflect.Type must be stored
	variadic bool
}

// TODO(crawshaw): as these funcTypeFixedN structs have no methods,
// they could be defined at runtime using the StructOf function.
type funcTypeFixed4 struct {
	funcType
	args [4]*rtype
}
type funcTypeFixed8 struct {
	funcType
	args [8]*rtype
}
type funcTypeFixed16 struct {
	funcType
	args [16]*rtype
}
type funcTypeFixed32 struct {
	funcType
	args [32]*rtype
}
type funcTypeFixed64 struct {
	funcType
	args [64]*rtype
}
type funcTypeFixed128 struct {
	funcType
	args [128]*rtype
}

// FuncOf is analogous to reflect.FuncOf.
func FuncOf(in []Type, out []Type, variadic bool) Type {
	nin := len(in)
	if variadic && (nin == 0 || in[nin-1] == nil ||
		(in[nin-1].kind() != kInvalid && in[nin-1].kind() != kSlice)) {

		panic("incomplete.FuncOf: last arg of variadic func must be slice")
	}
	if allTypesAreComplete(in) && allTypesAreComplete(out) {
		return Of(reflectFuncOf(in, out, variadic))
	}

	// Make a func type.
	var ifunc interface{} = (func())(nil)
	prototype := *(**funcType)(unsafe.Pointer(&ifunc))

	ft, args := makeFuncType(len(in) + len(out))
	*ft = *prototype
	ft.tflag = 0
	ft.ptrToThis = 0
	ft.inCount = uint16(len(in))
	ft.outCount = uint16(len(out))
	if variadic {
		ft.outCount |= funcOutCountVariadic
	}

	// TODO canonicalize return value
	return &itype{
		named:      nil,
		comparable: tfalse,
		iflag:      iflagSize,
		incomplete: &ft.rtype,
		info: &iFuncType{
			// safety: make a copy of in[] and out[]
			in:       append(([]Type)(nil), in...),
			out:      append(([]Type)(nil), out...),
			rargs:    args,
			variadic: variadic,
		},
	}
}

func reflectFuncOf(in []Type, out []Type, variadic bool) reflect.Type {
	rin := make([]reflect.Type, len(in))
	for i, t := range in {
		rin[i] = t.(*itype).complete
	}
	rout := make([]reflect.Type, len(out))
	for i, t := range out {
		rout[i] = t.(*itype).complete
	}
	return reflect.FuncOf(rin, rout, variadic)
}

func makeFuncType(n int) (ft *funcType, args []*rtype) {
	switch {
	case n == 0:
		args = nil
		ft = new(funcType)
	case n <= 4:
		fixed := new(funcTypeFixed4)
		args = fixed.args[:n]
		ft = &fixed.funcType
	case n <= 8:
		fixed := new(funcTypeFixed8)
		args = fixed.args[:n]
		ft = &fixed.funcType
	case n <= 16:
		fixed := new(funcTypeFixed16)
		args = fixed.args[:n]
		ft = &fixed.funcType
	case n <= 32:
		fixed := new(funcTypeFixed32)
		args = fixed.args[:n]
		ft = &fixed.funcType
	case n <= 64:
		fixed := new(funcTypeFixed64)
		args = fixed.args[:n]
		ft = &fixed.funcType
	case n <= 128:
		fixed := new(funcTypeFixed128)
		args = fixed.args[:n]
		ft = &fixed.funcType
	default:
		panic("incomplete.FuncOf: too many arguments")
	}
	return ft, args
}

func (info *iFuncType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "func("...)
	sep = ""
	for i, ityp := range info.in {
		if info.variadic && i == len(info.in)-1 {
			sep += "..."
		}
		dst = ityp.printTo(dst, sep)
		sep = ", "
	}
	switch len(info.out) {
	case 0:
		dst = append(dst, ')')
	case 1:
		dst = append(dst, ") "...)
	default:
		dst = append(dst, ") ("...)
	}
	sep = ""
	for _, ityp := range info.out {
		dst = ityp.printTo(dst, sep)
		sep = ", "
	}
	if len(info.out) > 1 {
		dst = append(dst, ')')
	}
	return dst
}

func (info *iFuncType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// functions always have known, fixed size
	return true
}

func (info *iFuncType) computeHashStr(t *itype) {

	// Build a hash and populate args slice
	args := info.rargs
	var hash uint32
	for i, in := range info.in {
		it := in.(*itype)
		computeHashStr(it)
		rt := it.incomplete
		args[i] = rt
		hash = fnv1(hash, byte(rt.hash>>24), byte(rt.hash>>16), byte(rt.hash>>8), byte(rt.hash))
	}
	if info.variadic {
		hash = fnv1(hash, 'v')
	}
	hash = fnv1(hash, '.')
	nin := len(info.in)
	for i, out := range info.out {
		it := out.(*itype)
		computeHashStr(it)
		rt := it.incomplete
		args[i+nin] = rt
		hash = fnv1(hash, byte(rt.hash>>24), byte(rt.hash>>16), byte(rt.hash>>8), byte(rt.hash))
	}
	/* TODO: needed?
	if len(args) > 50 {
		panic("reflect.FuncOf does not support more than 50 arguments")
	}
	*/
	t.incomplete.hash = hash
	t.incomplete.str = resolveReflectName(newName(t.string(), "", false))
}

func (info *iFuncType) completeType(t *itype) {
	t.complete = wrap(t.incomplete)
}
