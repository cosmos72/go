// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	"sync"
	"unsafe"
)

const maxStructFields = 128

// StructField is analogous to reflect.StructField,
// minus the Index and Offset fields.
type StructField struct {
	Name, PkgPath string
	Type          Type
	Tag           reflect.StructTag
	Anonymous     bool
}

type iStructType struct {
	fields []StructField
}

// The structLookupCache caches StructOf calls and canonicalizes their return values
var structLookupCache sync.Map // map[structCacheKey]*itype

// A structCacheKey is the key for use in the structLookupCache.
type structCacheKey struct {
	fields     [maxStructFields]StructField
	fieldCount uintptr
}

// StructOf is analogous to reflect.StructOf.
func StructOf(fields []StructField) Type {
	var (
		comparable = ttrue
		complete   = true
		pkgPath    = ""
		fs         = make([]structField, len(fields))
	)
	for i, field := range fields {
		if field.Name == "" {
			panic("incomplete.StructOf: field " + strconv.Itoa(i) + " has no name")
		}
		if !isValidFieldName(field.Name) {
			panic("incomplete.StructOf: field " + strconv.Itoa(i) + " has invalid name")
		}
		if field.Type == nil {
			panic("incomplete.StructOf: field " + strconv.Itoa(i) + " has no type")
		}
		if field.Type.(*itype).complete == nil {
			complete = false
			break
		}
		comparable = andTribool(comparable, field.Type.(*itype).comparable)

		f, fpkgPath := field.toRuntime()
		if fpkgPath != "" {
			if pkgPath == "" {
				pkgPath = fpkgPath
			} else if pkgPath != fpkgPath {
				panic("incomplete.StructOf: fields with different PkgPath " +
					pkgPath + " and " + fpkgPath)
			}
		}
		fs[i] = f
	}
	if complete {
		return Of(reflectStructOf(fields))
	}
	// safety: make a copy of fields[]
	ckey := structCacheKey{fieldCount: uintptr(len(fields))}
	copy(ckey.fields[:], fields)

	var ickey interface{} = ckey
	if t, ok := structLookupCache.Load(ickey); ok {
		return t.(Type)
	}

	// Make the struct type.
	// TODO: create wrapper methods for embedded fields
	var istruct interface{} = struct{}{}
	st := **(**structType)(unsafe.Pointer(&istruct))
	if pkgPath != "" {
		st.pkgPath = newName(pkgPath, "", false)
	}
	st.fields = fs

	t := &itype{
		named:      nil,
		comparable: comparable,
		iflag:      iflag(0),
		incomplete: &st.rtype,
		info: &iStructType{
			fields: ckey.fields[0:len(fields)],
		},
	}
	ret, _ := structLookupCache.LoadOrStore(ickey, t)
	return ret.(Type)
}

// StructField.toRuntime takes a StructField value passed to StructOf and
// returns both the corresponding internal representation, of type
// structField, and the pkgpath value to use for this field.
func (field *StructField) toRuntime() (structField, string) {
	if field.Anonymous && field.PkgPath != "" {
		panic("incomplete.StructOf: field \"" + field.Name + "\" is anonymous but has PkgPath set")
	}

	exported := field.PkgPath == ""
	if exported {
		// Best-effort check for misuse.
		// Since this field will be treated as exported, not much harm done if Unicode lowercase slips through.
		c := field.Name[0]
		if 'a' <= c && c <= 'z' || c == '_' {
			panic("incomplete.StructOf: field \"" + field.Name + "\" is unexported but missing PkgPath")
		}
	}

	offsetEmbed := uintptr(0)
	if field.Anonymous {
		offsetEmbed = 1
	}

	f := structField{
		name:        newName(field.Name, string(field.Tag), exported),
		typ:         nil, // will be set by iStructType.computeHashStr
		offsetEmbed: offsetEmbed,
	}
	return f, field.PkgPath
}

func reflectStructOf(fields []StructField) reflect.Type {
	rfields := make([]reflect.StructField, len(fields))
	for i, field := range fields {
		rfields[i] = field.toReflect()
	}
	return reflect.StructOf(rfields)
}

func (field *StructField) toReflect() reflect.StructField {
	return reflect.StructField{
		Name:      field.Name,
		PkgPath:   field.PkgPath,
		Type:      field.Type.(*itype).complete,
		Tag:       field.Tag,
		Offset:    0,
		Index:     nil,
		Anonymous: field.Anonymous,
	}
}

func (field *StructField) printTo(bytes []byte, separator string) []byte {
	return append(append(append(append(bytes,
		separator...), field.Name...), ' '), field.Type.string()...)
}

func (info *iStructType) printTo(dst []byte, sep string) []byte {
	dst = append(append(dst, sep...), "struct "...)
	sep = "{ "
	for i := range info.fields {
		dst = info.fields[i].printTo(dst, sep)
		sep = "; "
	}
	if len(info.fields) == 0 {
		dst = append(dst, "{}"...)
	} else {
		dst = append(dst, " }"...)
	}
	return dst
}

func (info *iStructType) computeSize(t *itype, work map[*itype]struct{}) bool {
	var size, lastzero uintptr
	var align, fieldAlign uint8
	ok := true
	for _, field := range info.fields {
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
	if ok {
		t.setSize(size, align, fieldAlign)
	}
	return ok
}

// align returns the result of rounding x up to a multiple of n.
// n must be a power of two.
// Must be kept in sync with reflect.align()
func doAlign(x, n uintptr) uintptr {
	return (x + n - 1) &^ (n - 1)
}

func (info iStructType) computeHashStr(t *itype) {
	panic("unimplemented")
}

func (info *iStructType) completeType(t *itype) {
	panic("unimplemented")
}
