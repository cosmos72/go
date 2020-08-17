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
	Name    string
	PkgPath string
	Type    Type // receiver = first arg, except for interface methods
	// Index int
}

func methodsFromReflect(rtyp reflect.Type) []Method {
	n := rtyp.NumMethod()
	if n == 0 {
		return nil
	}
	mtd := make([]Method, n)
	for i := 0; i < n; i++ {
		mtd[i].fromReflect(rtyp.Method(i))
	}
	return mtd
}

func (mtd *Method) fromReflect(rmethod reflect.Method) {
	mtd.Name = rmethod.Name
	mtd.PkgPath = rmethod.PkgPath
	mtd.Type = of(rmethod.Type)
	// mtd.Index = rmethod.Index
}

// print interface method
func (mtd *Method) printTo(dst []byte, sep string) []byte {
	dst = append(dst, sep...)
	if path := mtd.PkgPath; path != "" {
		dst = append(append(dst, filename(path)...), '.')
	}
	dst = append(append(dst, mtd.Name...), ' ')

	// omit "func" prefix in method type
	buf := mtd.Type.printTo(([]byte)(nil), "")
	if len(buf) >= 5 && string(buf[0:5]) == "func(" {
		buf = buf[4:]
	}
	return append(dst, buf...)
}

func completeMethods(t *itype) {
}
