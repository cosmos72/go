// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
)

type iFuncType struct {
	in       []Type
	out      []Type
	variadic bool
}

var rtypeFunc *rtype = unwrap(reflect.TypeOf(func() {}))

// FuncOf is analogous to reflect.FuncOf.
func FuncOf(in, out []Type, variadic bool) Type {
	nin := len(in)
	if variadic && (nin == 0 || in[nin-1] == nil ||
		(in[nin-1].kind() != kInvalid && in[nin-1].kind() != kSlice)) {

		panic("incomplete.FuncOf: last arg of variadic func must be slice")
	}
	if allTypesAreComplete(in) && allTypesAreComplete(out) {
		return Of(reflectFuncOf(in, out, variadic))
	}
	return &itype{
		named:      nil,
		comparable: tfalse,
		iflag:      iflagSize,
		incomplete: &rtype{
			size:       rtypeFunc.size,
			align:      rtypeFunc.align,
			fieldAlign: rtypeFunc.fieldAlign,
			kind:       kFunc,
		},
		info: iFuncType{
			// safety: make a copy of in[] and out[]
			in:       append(([]Type)(nil), in...),
			out:      append(([]Type)(nil), out...),
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

func (info iFuncType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "func("...)
	sep = ""
	for i, ityp := range info.in {
		if i == len(info.in)-1 && info.variadic {
			sep += "..."
		}
		dst = ityp.printTo(dst, sep)
		sep = ", "
	}
	dst = append(dst, ") "...)
	if len(info.out) > 1 {
		dst = append(dst, '(')
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

func (info iFuncType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// functions always have known, fixed size
	return true
}

func (info iFuncType) prepareRtype(t *itype) {
	panic("unimplemented")
}

func (info iFuncType) completeType(t *itype) {
	panic("unimplemented")
}
