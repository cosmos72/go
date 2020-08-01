// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

func push(t *itype, work map[*itype]struct{}) map[*itype]struct{} {
	if work == nil {
		work = make(map[*itype]struct{})
	} else if _, ok := work[t]; ok {
		panic("invalid Type loop detected: cannot compute size")
	}
	work[t] = struct{}{}
	return work
}

func computeSize(t *itype, work map[*itype]struct{}) {
	if t.iflag&iflagSize != 0 {
		return
	}
	push(t, work)
	align, fieldAlign := uint8(0), uint8(0)
	size, ok := uintptr(0), false
	switch t.kind() {
	case kInvalid:
		if u, _ := t.info.(*itype); u != nil {
			computeSize(u, work)
			if u.iflag&iflagSize != 0 {
				align, fieldAlign = u.align(), u.fieldAlign()
				size, ok = u.size(), true
			}
		}
	case kArray:
		a := t.info.(iArrayType)
		ielem := a.elem.(*itype)
		computeSize(ielem, work)
		if ielem.iflag&iflagSize != 0 {
			esize := ielem.size()
			if esize > 0 {
				max := ^uintptr(0) / esize
				if uintptr(a.count) > max {
					panic("incomplete.ArrayOf: array size would exceed virtual address space")
				}
			}
			align, fieldAlign = ielem.align(), ielem.fieldAlign()
			size, ok = uintptr(a.count)*esize, true
		}
	case kStruct:
		s := t.info.(iStructType)
		lastzero := uintptr(0)
		ok = true
		for _, field := range s.fields {
			ityp := field.Type.(*itype)
			computeSize(ityp, work)
			if ityp.iflag&iflagSize == 0 {
				ok = false
			} else if ok {
				fsize := ityp.size()
				falign := ityp.align()

				offset := doAlign(size, uintptr(falign))
				if falign > align {
					align = falign
				}
				size = offset + fsize
				if fsize == 0 {
					lastzero = size
				}
			}
		}
		fieldAlign = align
		if ok && size > 0 && lastzero == size {
			// This is a non-zero sized struct that ends in a
			// zero-sized field. We add an extra byte of padding,
			// to ensure that taking the address of the final
			// zero-sized field can't manufacture a pointer to the
			// next object in the heap. See issue 9401.
			size++
		}

	default:
		panic("internal error: Type has unknown size")
	}
	if ok {
		if t.incomplete == nil {
			t.incomplete = &rtype{}
		}
		t.incomplete.size = size
		t.incomplete.align = align
		t.incomplete.fieldAlign = fieldAlign
		t.iflag |= iflagSize
	}
	delete(work, t)
}

// align returns the result of rounding x up to a multiple of n.
// n must be a power of two.
// Must be kept in sync with reflect.align()
func doAlign(x, n uintptr) uintptr {
	return (x + n - 1) &^ (n - 1)
}
