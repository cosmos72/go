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

// ChanOf is analogous to reflect.ChanOf.
func ChanOf(dir reflect.ChanDir, elem Type) Type {
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ChanOf(dir, ielem.complete))
	}
	// Look in cache.
	ckey := cacheKey{kChan, ielem, nil, uintptr(dir)}
	if ret, ok := lookupCache.Load(ckey); ok {
		return ret.(Type)
	}

	// Make a channel type.
	var ichan interface{} = (chan unsafe.Pointer)(nil)
	ch := **(**chanType)(unsafe.Pointer(&ichan))
	ch.tflag = tflagRegularMemory
	ch.dir = uintptr(dir)
	ch.elem = nil

	return canonicalize(ckey, &itype{
		named:      nil,
		comparable: ttrue,
		iflag:      iflagSize,
		incomplete: &ch.rtype,
		info: &iChanType{
			elem: elem,
			dir:  dir,
		},
	})
}

func (info *iChanType) printTo(dst []byte, sep string) []byte {
	prefix := "chan "
	if info.dir == reflect.RecvDir {
		prefix = "<-chan "
	} else if info.dir == reflect.SendDir {
		prefix = "chan<- "
	}
	dst = append(append(dst, sep...), prefix...)
	return info.elem.printTo(dst, "")
}

func (info *iChanType) computeSize(t *itype, work map[*itype]struct{}) bool {
	// channels always have known, fixed size
	return true
}

func (info *iChanType) computeHashStr(t *itype) {
	ielem := info.elem.(*itype)
	computeHashStr(ielem)

	ch := (*chanType)(unsafe.Pointer(t.incomplete))
	ch.str = resolveReflectName(newName(t.string(), "", false))
	ch.hash = fnv1(ielem.incomplete.hash, 'c', byte(info.dir))

	ch.elem = ielem.incomplete
}

func (info *iChanType) completeType(t *itype) {
	t.complete = wrap(t.incomplete)
}
