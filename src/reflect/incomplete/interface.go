// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"unsafe"
)

type interfaceType struct {
	embedded []Type
	methods  []Method
}

const sizeOfInterface = unsafe.Sizeof((interface{})(nil))

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
		size:    sizeOfInterface,
		kind:    kInterface,
		tflag:   tflagSize,
		info: interfaceType{
			// safety: make a copy of embedded[]
			embedded: append(([]Type)(nil), embedded...),
		},
	}
}
