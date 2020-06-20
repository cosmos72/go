// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect

import (
	"sync"
	"unsafe"
)

// wrapperType represents a named type created at runtime.
// to allow creating recursive types, its underlying type is set after creation
type wrapperType struct {
	rtype
	uncommon   uncommonType
	underlying *rtype
}

// if t is part of a wrapperType created by NewNamed, return its underlying type.
// otherwise return t.
func unwrap(t *rtype, operation string) *rtype {
	if t.tflag&tflagWrapper == 0 {
		return t
	}
	w := (*wrapperType)(unsafe.Pointer(t))
	if w.underlying == nil || w.kind != w.underlying.kind {
		panic("reflect: " + operation + " of incomplete Type " + w.String())
	}
	return w.underlying
}

// NewNamed returns a new named type with the given pkgPath and name.
//
// The returned type is incomplete until SetUnderlying is called on it.
func NewNamed(pkgPath string, name string) Type {
	if name == "" {
		panic("reflect.NewNamed: name is empty")
	}
	if !isValidFieldName(name) {
		panic("reflect.NewNamed: name is invalid")
	}
	str := name
	var pktPathNameOff nameOff
	if pkgPath != "" {
		pktPathNameOff = resolveReflectName(newName(pkgPath, "", false))
		i := len(pkgPath) - 1
		for i >= 0 && pkgPath[i] != '.' {
			i--
		}
		str = pkgPath[i:] + "." + name
	}
	wrapper := &wrapperType{
		rtype: rtype{
			size:       0, // not known yet
			ptrdata:    0, // not known yet
			hash:       0, // set below
			tflag:      tflagNamed | tflagUncommon | tflagUnknownSize | tflagIncomplete | tflagWrapper,
			align:      0,              // not known yet
			fieldAlign: 0,              // not known yet
			kind:       uint8(Invalid), // not known yet
			equal:      nil,            // not known yet
			gcdata:     nil,            // not known yet
			str:        resolveReflectName(newName(str, "", true)),
			ptrToThis:  0,
		},
		uncommon: uncommonType{
			pkgPath: pktPathNameOff,
			mcount:  0,
			xcount:  0,
			moff:    0,
		},
		underlying: nil,
	}
	// this is a new unique type, any hash would be ok
	wrapper.hash = fnv1(uint32(uintptr(unsafe.Pointer(&wrapper.rtype))), []byte(str)...)
	return &wrapper.rtype
}

// SetUnderlying sets the underlying type of a named type created with NewNamed
// and completes it.
func SetUnderlying(named Type, underlying Type) {
	t := named.(*rtype)
	if t.tflag&tflagWrapper == 0 {
		panic("reflect: SetUnderlying of Type not created with NewNamed: " + t.String())
	}
	w := (*wrapperType)(unsafe.Pointer(t))
	if t.tflag&tflagIncomplete == 0 /* || w.underlying != nil || t.Kind() != Invalid */ {
		panic("reflect: SetUnderlying already invoked on Type " + t.String())
	}

	u := underlying.(*rtype)
	if u.tflag&tflagIncomplete != 0 {
		panic("reflect: SetUnderlying: underlying Type " + t.String() + " is incomplete")
	}
	if u.tflag&tflagWrapper != 0 {
		u = (*wrapperType)(unsafe.Pointer(t)).underlying
	}
	w.size = u.size
	w.ptrdata = u.ptrdata
	w.tflag = (w.tflag | u.tflag&tflagRegularMemory) &^ (tflagUnknownSize | tflagIncomplete)
	w.align = u.align
	w.fieldAlign = u.fieldAlign
	w.kind = u.kind
	w.equal = u.equal
	w.gcdata = u.gcdata
	w.underlying = u
}

type (
	wrapperSet map[*wrapperType]struct{}
	rtypeSet   map[*rtype]struct{}

	rtypeMap   map[*rtype]wrapperSet
	wrapperMap map[*wrapperType]rtypeSet
)

var (
	// protects incompleteTypes and incompleteWrappers for concurrent access
	incompleteMutex sync.Mutex

	// map each incomplete *rtype to its dependencies
	incompleteTypes = make(rtypeMap)

	// map each incomplete *wrapperType to the *rtypes that depend on it
	incompleteWrappers = make(wrapperMap)
)

func (dst wrapperSet) union(src wrapperSet) {
	for k, v := range src {
		dst[k] = v
	}
}

func (dst rtypeSet) insert(t *rtype) {
	dst[t] = struct{}{}
}

func (dst wrapperMap) insert(wrapper *wrapperType, t *rtype) {
	set, ok := dst[wrapper]
	if !ok {
		set = make(rtypeSet)
		dst[wrapper] = set
	}
	set.insert(t)
}

func markIncomplete(t *rtype, directDependencies ...*rtype) {
	t.tflag |= tflagIncomplete
	deps := make(wrapperSet)

	incompleteMutex.Lock()
	for _, etyp := range directDependencies {
		deps.union(incompleteTypes[etyp])
	}
	incompleteTypes[t] = deps
	for wrapper := range deps {
		incompleteWrappers.insert(wrapper, t)
	}
	incompleteMutex.Unlock()
}

// complete the specified type.
func complete(t *rtype) {
	switch t.Kind() {
	case Array:
		completeArray(t)
	default:
		return
	}

	incompleteMutex.Lock()
	set := incompleteTypes[t]
	delete(incompleteTypes, t)
	for wrapper := range set {
		delete(incompleteWrappers[wrapper], t)
	}
	incompleteMutex.Unlock()
}
