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
	if t.incomplete == nil {
		t.incomplete = &rtype{}
	}
	switch t.kind() {
	case kInvalid:
		if u, _ := t.info.(*itype); u != nil {
			computeSize(u, work)
			if u.iflag&iflagSize != 0 {
				t.incomplete.size = u.incomplete.size
				t.iflag |= iflagSize
			}
		}
	case kArray:
		a := t.info.(iArrayType)
		ielem := a.elem.(*itype)
		computeSize(ielem, work)
		if ielem.iflag&iflagSize != 0 {
			t.incomplete.size = uintptr(a.count) * ielem.size()
			t.iflag |= iflagSize
		}
	case kStruct:
		s := t.info.(iStructType)
		size := uintptr(0)
		ok := true
		for _, field := range s.fields {
			ityp := field.Type.(*itype)
			computeSize(ityp, work)
			if ityp.iflag&iflagSize != 0 {
				// TODO also consider alignment
				size += ityp.size()
			} else {
				ok = false
			}
		}
		if ok {
			t.incomplete.size = size
			t.iflag |= iflagSize
		}
	default:
		panic("internal error: Type has unknown size")
	}
	delete(work, t)
}
