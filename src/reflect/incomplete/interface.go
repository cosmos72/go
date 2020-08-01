// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
)

type iInterfaceType struct {
	embedded []Type
}

var rtypeInterface *rtype = unwrap(reflect.TypeOf((*interface{})(nil)).Elem())

// InterfaceOf returns an incomplete interface type with the given list of
// named interface types. InterfaceOf panics if one of the given embedded types
// is unnamed or its kind is not reflect.Interface. It also panics if types
// with distinct, non-empty package paths are embedded.
//
// Explicit methods can be added with AddMethod.
func InterfaceOf(embedded []Type) Type {
	return &itype{
		named:  nil,
		method: nil,
		iflag:  iflagSize,
		incomplete: &rtype{
			kind:       kInterface,
			size:       rtypeInterface.size,
			align:      rtypeInterface.align,
			fieldAlign: rtypeInterface.fieldAlign,
		},
		info: iInterfaceType{
			// safety: make a copy of embedded[]
			embedded: append(([]Type)(nil), embedded...),
		},
	}
}
