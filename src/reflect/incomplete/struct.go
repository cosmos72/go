// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
)

// StructField is analogous to reflect.StructField, minus the Index and Offset
// fields.
type StructField struct {
	Name, PkgPath string
	Type          Type
	Tag           reflect.StructTag
	Anonymous     bool
}

type iStructType struct {
	fields []StructField
}

// StructOf is analogous to reflect.StructOf.
func StructOf(fields []StructField) Type {
	comparable := ttrue
	complete := true
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

	}
	if complete {
		return Of(reflectStructOf(fields))
	}
	return &itype{
		named:      nil,
		comparable: comparable,
		iflag:      iflag(0),
		incomplete: &rtype{
			kind: kStruct,
		},
		info: iStructType{
			// safety: make a copy of fields[]
			fields: append(([]StructField)(nil), fields...),
		},
	}
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

func (info iStructType) printTo(dst []byte, sep string) []byte {
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

func (info iStructType) completeType(t *itype) {
	panic("unimplemented")
}
