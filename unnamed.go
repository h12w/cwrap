// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type Num struct {
	SimpleConv
}

func (n *Num) SetGoName(name string) {
	if s, ok := n.SimpleConv.Type.(GoNameSetter); ok {
		s.SetGoName(name)
	}
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
	elementType Type
	length      int
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

func (s *Slice) Size() int {
	return MachineSize
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
	conv(w, ":", "nil", c, s.CgoName())
	fp(w, "if len(", g, ")>0 {")
	convPtr(w, "", "&"+g+"[0]", c, s.CgoName())
	fp(w, "}")
}

type SliceSlice struct {
	slice Slice
}

func (s *SliceSlice) Size() int {
	return MachineSize
}

func (s *SliceSlice) GoName() string {
	return "[]" + s.slice.GoName()
}

func (s *SliceSlice) CgoName() string {
	return "*" + s.slice.CgoName()
}

func (s *SliceSlice) WriteSpec(w io.Writer) {
	fpn(w, s.GoName())
}

func (s *SliceSlice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for SliceSlice yet.")
}

func (s *SliceSlice) ToCgo(w io.Writer, assign, g, c string) {
	c_ := c + "_"
	fp(w, c_, " := make([]", s.slice.CgoName(), ", len(", g, "))")
	fp(w, "for i := range ", g, "{")
	fp(w, "if len(", g, "[i])>0 {")
	convPtr(w, "", "&"+g+"[i][0]", c_+"[i]", s.slice.CgoName())
	fp(w, "}")
	fp(w, "}")
	(&Slice{&baseType{"[]*" + s.slice.GoName(), s.slice.CgoName(), MachineSize}}).ToCgo(w, assign, c_, c)
}

type StringSlice struct {
	baseType
}

func newStringSlice() *StringSlice {
	return &StringSlice{baseType{
		"[]string",
		"**C.char",
		MachineSize,
	}}
}

func (s *StringSlice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for StringSlice yet.")
}

func (s *StringSlice) ToCgo(w io.Writer, assign, g, c string) {
	c_ := c + "_"
	fp(w, c_, " := make([]*C.char, len(", g, "))")
	fp(w, "for i := range ", g, "{")
	newString().ToCgo(w, "", g+"[i]", c_+"[i]")
	fp(w, "}")
	(&Slice{&baseType{"[]string", "*C.char", MachineSize}}).ToCgo(w, assign, c_, c)
}

type String struct {
	baseType
}

func newString() *String {
	return &String{baseType{
		"string",
		"*C.char",
		MachineSize,
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
	pointedType Type
}

func (r *ReturnPtr) Size() int {
	return MachineSize
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
	pointedType Type
}

func (t *Ptr) Size() int {
	return MachineSize
}

func (t *Ptr) GoName() string {
	n := t.pointedType.GoName()
	if n == "" || n == "[0]byte" {
		return "uintptr"
	}
	return "*" + n
}

func (t *Ptr) CgoName() string {
	n := t.pointedType.CgoName()
	if n == "" {
		return "unsafe.Pointer"
	}
	return "*" + n
}

func (t *Ptr) WriteSpec(w io.Writer) {
	fpn(w, t.GoName())
}

func (t *Ptr) ToCgo(w io.Writer, assign, g, c string) {
	if t.CgoName() == "unsafe.Pointer" {
		conv(w, assign, g, c, t.CgoName())
	} else {
		convPtr(w, assign, g, c, t.CgoName())
	}
}

func (t *Ptr) ToGo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, c, g, t.GoName())
}

type Void struct {
	baseType
}

func (*Void) ToGo(w io.Writer, assign, g, c string) {
	panic("should not goes here.")
}

func (*Void) ToCgo(w io.Writer, assign, g, c string) {
	panic("should not goes here.")
}
