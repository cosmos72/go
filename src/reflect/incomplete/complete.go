// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package incomplete

import (
	"reflect"
)

// Complete completes the incomplete types in in, transforming them to a list
// of reflect.Type types. The function method is called once for each method
// added with AddMethod and should return an implementation of that method:
// a function whose first argument is the receiver.
// The list out contains the fully usable resulting types, except that methods
// can be called on them only after Complete has returned. The index indicates
// which type will be the method receiver, and stub indicates the method.
func Complete(
	in []Type,
	method func(out []reflect.Type, index int, stub Method) interface{},
) []reflect.Type {

	panic("unimplemented: incomplete.Complete()")
}
