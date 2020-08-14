// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	"unsafe"
)

type iArrayType struct {
	elem  Type
	count int
}

// maxPtrmaskBytes must be kept in sync with ../type.go:/^maxPtrmaskBytes.
// See cmd/compile/internal/gc/reflect.go for derivation of constant.
const maxPtrmaskBytes = 2048

// ArrayOf creates an incomplete array type with the given count and
// element type described by elem.
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("incomplete.ArrayOf: element count is negative")
	}
	ielem := elem.(*itype)
	if ielem.complete != nil {
		return Of(reflect.ArrayOf(count, ielem.complete))
	}
	// Look in cache.
	ckey := cacheKey{kArray, ielem, nil, uintptr(count)}
	if ret, ok := lookupCache.Load(ckey); ok {
		return ret.(Type)
	}

	var iarray interface{} = [1]unsafe.Pointer{}
	array := **(**arrayType)(unsafe.Pointer(&iarray))
	array.elem = nil
	array.ptrToThis = 0
	array.len = uintptr(count)

	return canonical(ckey, &itype{
		named:      nil,
		comparable: ielem.comparable,
		iflag:      ielem.iflag & iflagSize,
		incomplete: &array.rtype,
		info: &iArrayType{
			elem:  elem,
			count: count,
		},
	})
}

func (info *iArrayType) printTo(dst []byte, sep string) []byte {
	dst = append(append(append(append(
		dst, sep...), '['), strconv.Itoa(info.count)...), ']')
	return info.elem.printTo(dst, "")
}

func (info *iArrayType) computeSize(t *itype, work map[*itype]struct{}) bool {
	ielem := info.elem.(*itype)
	if !ielem.computeSize(ielem, work) {
		return false
	}
	esize := ielem.size()
	if esize > 0 {
		max := ^uintptr(0) / esize
		if uintptr(info.count) > max {
			panic("incomplete.ArrayOf: array size would exceed virtual address space")
		}
	}
	t.setSize(uintptr(info.count)*esize, ielem.align(), ielem.fieldAlign())
	info.computePtrData(t)
	return true
}

func (info *iArrayType) computePtrData(t *itype) {
	count := info.count
	array := (*arrayType)(unsafe.Pointer(t.incomplete))

	ielem := info.elem.(*itype)
	relem := ielem.incomplete
	esize := ielem.size()

	if info.count > 0 && relem.ptrdata != 0 {
		array.ptrdata = esize*uintptr(count-1) + relem.ptrdata
	}

	switch {
	case relem.ptrdata == 0 || array.size == 0:
		// No pointers.
		array.gcdata = nil
		array.ptrdata = 0

	case count == 1:
		// In memory, 1-element array looks just like the element.
		array.kind |= relem.kind & kindGCProg
		array.gcdata = relem.gcdata
		array.ptrdata = relem.ptrdata

	case relem.kind&kindGCProg == 0 && array.size <= maxPtrmaskBytes*8*ptrSize:
		// Element is small with pointer mask; array is still small.
		// Create direct pointer mask by turning each 1 bit in elem
		// into count 1 bits in larger mask.
		mask := make([]byte, (array.ptrdata/ptrSize+7)/8)
		emitGCMask(mask, 0, relem, array.len)
		array.gcdata = &mask[0]

	default:
		// Create program that emits one element
		// and then repeats to make the array.
		prog := []byte{0, 0, 0, 0} // will be length of prog
		prog = appendGCProg(prog, relem)
		// Pad from ptrdata to size.
		elemPtrs := relem.ptrdata / ptrSize
		elemWords := relem.size / ptrSize
		if elemPtrs < elemWords {
			// Emit literal 0 bit, then repeat as needed.
			prog = append(prog, 0x01, 0x00)
			if elemPtrs+1 < elemWords {
				prog = append(prog, 0x81)
				prog = appendVarint(prog, elemWords-elemPtrs-1)
			}
		}
		// Repeat count-1 times.
		if elemWords < 0x80 {
			prog = append(prog, byte(elemWords|0x80))
		} else {
			prog = append(prog, 0x80)
			prog = appendVarint(prog, elemWords)
		}
		prog = appendVarint(prog, uintptr(count)-1)
		prog = append(prog, 0)
		*(*uint32)(unsafe.Pointer(&prog[0])) = uint32(len(prog) - 4)
		array.kind |= kindGCProg
		array.gcdata = &prog[0]
		array.ptrdata = array.size // overestimate but ok; must match program
	}
}

func (info *iArrayType) computeHashStr(t *itype) {
	panic("unimplemented")
}

func (info *iArrayType) completeType(t *itype) {
	panic("unimplemented")
}
