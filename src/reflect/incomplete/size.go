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
	if t.tflag&tflagSize != 0 {
		return
	}
	push(t, work)
	switch t.kind {
	case kInvalid:
		if u, _ := t.info.(*itype); u != nil {
			computeSize(u, work)
			if u.tflag&tflagSize != 0 {
				t.size = u.size
				t.tflag |= tflagSize
			}
		}
	case kArray:
		a := t.info.(arrayType)
		ielem := a.elem.(*itype)
		computeSize(ielem, work)
		if ielem.tflag&tflagSize != 0 {
			t.size = uintptr(a.count) * ielem.size
			t.tflag |= tflagSize
		}
	case kStruct:
		s := t.info.(structType)
		size := uintptr(0)
		ok := true
		for _, field := range s.fields {
			ityp := field.Type.(*itype)
			computeSize(ityp, work)
			if ityp.tflag&tflagSize != 0 {
				size += ityp.size
			} else {
				ok = false
			}
		}
		if ok {
			t.size = size
			t.tflag |= tflagSize
		}
	default:
		panic("internal error: Type has unknown size")
	}
	delete(work, t)
}
