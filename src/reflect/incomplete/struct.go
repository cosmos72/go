// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
	"strconv"
	_ "unsafe" // needed by go:linkname
)

type iStructType struct {
	fields []StructField
}

// StructField is analogous to reflect.StructField, minus the Index and Offset
// fields.
type StructField struct {
	Name, PkgPath string
	Type          Type
	Tag           reflect.StructTag
	Anonymous     bool
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

// StructOf is analogous to reflect.StructOf.
func StructOf(fields []StructField) Type {
	var str = []byte("struct ")
	sep := "{ "
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
		str = append(str, sep...)
		str = append(str, field.Name...)
		str = append(str, ' ')
		str = append(str, field.Type.string()...)
		sep = "; "
	}
	if complete {
		return Of(reflectStructOf(fields))
	}
	if len(fields) == 0 {
		str = append(str, "{}"...)
	} else {
		str = append(str, " }"...)
	}
	return &itype{
		named:   nil,
		methods: nil,
		str:     string(str),
		iflag:   iflag(0),
		incomplete: &rtype{
			kind: kStruct,
		},
		info: iStructType{
			// safety: make a copy of fields[]
			fields: append(([]StructField)(nil), fields...),
		},
	}
}

//go:linkname isValidFieldName reflect.isValidFieldName
func isValidFieldName(fieldName string) bool

func reflectStructOf(fields []StructField) reflect.Type {
	rfields := make([]reflect.StructField, len(fields))
	for i, field := range fields {
		rfields[i] = field.toReflect()
	}
	return reflect.StructOf(rfields)
}
