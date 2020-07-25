// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import "reflect"

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
		Type:      field.Type.(*itype).extra.(reflect.Type),
		Tag:       field.Tag,
		Offset:    0,
		Index:     nil,
		Anonymous: field.Anonymous,
	}
}

// StructOf is analogous to reflect.StructOf.
func StructOf(fields []StructField) Type {
	ok := true
	for _, field := range fields {
		ityp := field.Type.(*itype)
		if ityp.tflag&tflagRType == 0 {
			ok = false
			break
		}
	}
	if ok {
		return Of(reflectStructOf(fields))
	}
	return &itype{
		named:   nil,
		methods: nil,
		size:    0,
		kind:    kStruct,
		tflag:   tflag(0),
		extra: structType{
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
