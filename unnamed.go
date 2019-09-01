// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type Void struct {
}

func (v *Void) WriteSpec(w io.Writer) {
	fp(w, "nil")
}

func (v *Void) Size() int {
	return 0
}

func (v *Void) GoName() string {
	return "int /* void */"
}

func (v *Void) CgoName() string {
	return "/* void */"
}

func (v *Void) ToCgo(w io.Writer, assign, g, c string) {
}

func (v *Void) ToGo(w io.Writer, assign, g, c string) {
}

type Num struct {
	baseEqualType
}

type Bool struct {
	baseType
}

func (n *Bool) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, ":", "0", c, n.CgoName())
	fp(w, "if ", g, " {")
	conv(w, "", "1", c, n.CgoName())
	fp(w, "}")
}

func (n *Bool) ToGo(w io.Writer, assign, g, c string) {
	conv(w, assign, c+"==1", g, n.GoName())
}

type Array struct {
	elementType EqualType
	length      int
}

func (a *Array) ToCgo(w io.Writer, assign, g, c string) {
	convValue(w, assign, g, c, a.CgoName())
}

func (a *Array) ToGo(w io.Writer, assign, g, c string) {
	convValue(w, assign, c, g, a.GoName())
}

func (a *Array) Size() int {
	return a.elementType.Size() * a.length
}

func (a *Array) GoName() string {
	return writeToString(a.WriteSpec)
}

func (a *Array) WriteSpec(w io.Writer) {
	fpn(w, "[", a.length, "]", a.elementType.GoName())
}

func (a *Array) CgoName() string {
	return sprint("[", a.length, "]", a.elementType.CgoName())
}

type Slice struct {
	elementType Type
}

func (s *Slice) GoName() string {
	return "[]" + s.elementType.GoName()
}

func (s *Slice) CgoName() string {
	return "*" + s.elementType.CgoName()
}

func (s *Slice) WriteSpec(w io.Writer) {
	fpn(w, s.GoName())
}

func (s *Slice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for Slice yet.")
}

func (s *Slice) ToCgo(w io.Writer, assign, g, c string) {
	if assign == ":" {
		conv(w, ":", "nil", c, s.CgoName())
	}
	fp(w, "if len(", g, ")>0 {")
	convPtr(w, "", "&"+g+"[0]", c, s.CgoName())
	fp(w, "}")
}

type SliceSlice struct {
	Slice
}

func newSliceSlice(elementType Type) *SliceSlice {
	return &SliceSlice{Slice{&Slice{elementType}}}
}

func (s *SliceSlice) ToCgo(w io.Writer, assign, g, c string) {
	c_ := c + "_"
	fp(w, c_, " := make([]", s.elementType.CgoName(), ", len(", g, "))")
	fp(w, "for i := range ", g, "{")
	s.elementType.ToCgo(w, "", g+"[i]", c_+"[i]")
	fp(w, "}")
	s.Slice.ToCgo(w, assign, c_, c)
}

func newStringSlice() *SliceSlice {
	return &SliceSlice{Slice{newString()}}
}

type String struct {
	baseType
}

func newString() *String {
	return &String{baseType{
		"string",
		"*C.char",
	}}
}

func (s *String) ToCgo(w io.Writer, assign, g, c string) {
	fp(w, c, assign, "=C.CString(", g, ")")
	fp(w, "defer C.free(unsafe.Pointer(", c, "))")
}

func (s *String) ToGo(w io.Writer, assign, g, c string) {
	fp(w, g, assign, "=C.GoString(", c, ")")
}

type ReturnPtr struct {
	pointedType EqualType
}

func (r *ReturnPtr) GoName() string {
	return r.pointedType.GoName()
}

func (r *ReturnPtr) CgoName() string {
	return "*" + r.pointedType.CgoName()
}

func (r *ReturnPtr) WriteSpec(w io.Writer) {
	fpn(w, r.GoName())
}

func (r *ReturnPtr) ToGo(w io.Writer, assign, g, c string) {
}

func (r *ReturnPtr) ToCgo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, "&"+g, c, r.CgoName())
}

type Ptr struct {
	pointedType EqualType
}

func (t *Ptr) Size() int {
	return MachineSize
}

func (t *Ptr) GoName() string {
	if t.pointedType.Size() == 0 || t.pointedType.GoName() == "" {
		return "uintptr"
	}
	return "*" + t.pointedType.GoName()
}

func (t *Ptr) CgoName() string {
	if t.pointedType.Size() == 0 || t.pointedType.CgoName() == "" {
		return "unsafe.Pointer"
	}
	return "*" + t.pointedType.CgoName()
}

func (t *Ptr) WriteSpec(w io.Writer) {
	fpn(w, t.GoName())
}

func (t *Ptr) ToCgo(w io.Writer, assign, g, c string) {
	if t.CgoName() == "unsafe.Pointer" || IsFunc(t.pointedType) {
		conv(w, assign, g, c, t.CgoName())
	} else {
		convPtr(w, assign, g, c, t.CgoName())
	}
}

func (t *Ptr) ToGo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, c, g, t.GoName())
}

type FuncType struct {
	baseEqualType
}

func newFuncType() *FuncType {
	return &FuncType{baseEqualType{"[0]byte", "[0]byte", 0, nil}}
}
