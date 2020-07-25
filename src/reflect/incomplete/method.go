// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
)

// Method represents an incomplete method.
// Unlike in reflect.Method, the implementing Func is not part of this
// structure.
type Method struct {
	Name, PkgPath string
	Type          Type // receiver = first arg, except for interface methods
	Index         int
}

func methodsFromReflect(rtyp reflect.Type) *[]Method {
	if rtyp.Kind() != reflect.Ptr && rtyp.Kind() != reflect.Interface {
		// TODO distinguish methods with value receiver and pointer receiver
		rtyp = reflect.PtrTo(rtyp)
	}
	n := rtyp.NumMethod()
	if n == 0 {
		return nil
	}
	mtd := make([]Method, n)
	for i := 0; i < n; i++ {
		mtd[i].fromReflect(rtyp.Method(i))
	}
	return &mtd
}

func (mtd *Method) fromReflect(rmethod reflect.Method) {
	mtd.Name = rmethod.Name
	mtd.PkgPath = rmethod.PkgPath
	mtd.Type = Of(rmethod.Type) // TODO avoid infinite recursion
	mtd.Index = rmethod.Index
}
