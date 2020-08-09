// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"unsafe"
)

type iChanType struct {
	elem Type
	dir  reflect.ChanDir
}

var rtypeChan *rtype = unwrap(reflect.TypeOf(make(chan unsafe.Pointer)))

// ChanOf is analogous to reflect.ChanOf.
func ChanOf(dir reflect.ChanDir, elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ChanOf(dir, ielem.complete))
	}
	incomplete := *rtypeChan
	return &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &incomplete,
		info: iChanType{
			elem: elem,
			dir:  dir,
		},
	}
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

func (info iChanType) completeType(t *itype) {
	panic("unimplemented")
}
