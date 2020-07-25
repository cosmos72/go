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
