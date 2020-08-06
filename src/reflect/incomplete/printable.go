// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
)

type printable interface {
	printTo(dst []byte, separator string) []byte
}

func (t *itype) printTo(bytes []byte, separator string) []byte {
	bytes = append(bytes, separator...)
	if t.complete != nil {
		return append(bytes, t.complete.String()...)
	} else if t.named != nil {
		return append(bytes, t.named.str...)
	} else if t.info != nil {
		return t.info.printTo(bytes, "")
	} else {
		panic("reflect/incomplete error: Type string representation should be known, but it is not")
	}
}

func (info iArrayType) printTo(dst []byte, sep string) []byte {
	dst = append(append(append(append(
		dst, sep...), '['), strconv.Itoa(info.count)...), ']')
	return info.elem.printTo(dst, "")
}

func (info iChanType) printTo(dst []byte, sep string) []byte {
	prefix := "chan "
	if info.dir == reflect.RecvDir {
		prefix = "<-chan "
	} else if info.dir == reflect.SendDir {
		prefix = "chan<- "
	}
	dst = append(append(dst, sep...), prefix...)
	return info.elem.printTo(dst, "")
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

func (info iMapType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "map["...)
	dst = info.key.printTo(dst, "")
	dst = append(dst, ']')
	return info.elem.printTo(dst, "")
}

func (info iInterfaceType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "interface{"...)

	if len(info.allMethod) == 0 {
		return append(dst, '}')
	}
	sep = " "
	for i := range info.allMethod {
		info.allMethod[i].printTo(dst, sep)
		sep = "; "
	}
	return append(dst, " }"...)
}

func (info iPtrType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), '*')
	return info.elem.printTo(dst, "")
}

func (info iSliceType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "[]"...)
	return info.elem.printTo(dst, "")
}
