// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type funcType struct {
	in       []Type
	out      []Type
	variadic bool
}

const sizeOfFunc = unsafe.Sizeof(func() {})

// FuncOf is analogous to reflect.FuncOf.
func FuncOf(in, out []Type, variadic bool) Type {
	nin := len(in)
	if variadic && (nin == 0 ||
		(in[nin-1].(*itype).kind != kInvalid &&
			in[nin-1].(*itype).kind != kSlice)) {

		panic("incomplete.FuncOf: last arg of variadic func must be slice")
	}
	if allTypesHaveReflectType(in) && allTypesHaveReflectType(out) {
		return Of(reflectFuncOf(in, out, variadic))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    sizeOfFunc,
		kind:    kFunc,
		tflag:   tflagSize,
		extra: funcType{
			// safety: make a copy of in[] and out[]
			in:       append(([]Type)(nil), in...),
			out:      append(([]Type)(nil), out...),
			variadic: variadic,
		},
	}
}

func allTypesHaveReflectType(types []Type) bool {
	for _, t := range types {
		if t.(*itype).tflag&tflagRType == 0 {
			return false
		}
	}
	return true
}

func reflectFuncOf(in []Type, out []Type, variadic bool) reflect.Type {
	rin := make([]reflect.Type, len(in))
	for i, t := range in {
		rin[i] = t.(*itype).extra.(reflect.Type)
	}
	rout := make([]reflect.Type, len(out))
	for i, t := range out {
		rout[i] = t.(*itype).extra.(reflect.Type)
	}
	return reflect.FuncOf(rin, rout, variadic)
}
