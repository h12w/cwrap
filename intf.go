// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type CNamer interface {
	Id() string
	CName() string
	File() string
}

type GoNamer interface {
	GoName() string
}

type GoNameSetter interface {
	SetGoName(string)
}

type CgoNamer interface {
	CgoName() string
}

type SpecWriter interface {
	WriteSpec(w io.Writer)
}

type MethodsWriter interface {
	WriteMethods(w io.Writer)
}

type Decl interface {
	CNamer
	GoNamer
	SpecWriter
}

type TypeDecl interface {
	CNamer
	Type
	GoNameSetter
	MethodsWriter
}

type Type interface {
	GoNamer
	CgoNamer
	Size() int
	SpecWriter
}

type Conv interface {
	Type
	ToCgo(w io.Writer, assign, g, c string)
	ToGo(w io.Writer, assign, g, c string)
}

type NameOptimizer interface {
	OptimizeNames()
}

type baseType struct {
	goName  string
	cgoName string
	size    int
}

func (t *baseType) Size() int {
	return t.size
}

func (t *baseType) SetGoName(n string) {
	t.goName = n
}

func (t *baseType) GoName() string {
	if t.goName == "" {
		return sprint("[", t.size, "]byte")
	}
	return t.goName
}

func (t *baseType) CgoName() string {
	return t.cgoName
}

func (t *baseType) WriteSpec(w io.Writer) {
	fpn(w, t.goName)
}

type baseCNamer struct {
	id    string
	cName string
	file  string
}

func (e baseCNamer) Id() string {
	return e.id
}

func (e baseCNamer) CName() string {
	return e.cName
}

func (e baseCNamer) File() string {
	return e.file
}

type SimpleConv struct {
	Type
}

func (n SimpleConv) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, c, n.CgoName())
}

func (n SimpleConv) ToGo(w io.Writer, assign, g, c string) {
	conv(w, assign, c, g, n.GoName())
}

type ValueConv struct {
	Type
}

func (a ValueConv) ToCgo(w io.Writer, assign, g, c string) {
	convValue(w, assign, g, c, a.CgoName())
}

func (a ValueConv) ToGo(w io.Writer, assign, g, c string) {
	convValue(w, assign, c, g, a.GoName())
}

func conv(w io.Writer, assign, src, dst, dstType string) {
	if hasPrefix(dstType, "*") {
		dstType = "(" + dstType + ")"
	}
	fp(w, dst, assign, "=", dstType, "(", src, ")")
}

func convPtr(w io.Writer, assign, src, dst, dstType string) {
	fp(w, dst, assign, "=(", dstType, ")(unsafe.Pointer(", src, "))")
}

func convValue(w io.Writer, assign, src, dst, dstType string) {
	fp(w, dst, assign, "=*(*", dstType, ")(unsafe.Pointer(&", src, "))")
}

func IsEnum(v interface{}) bool {
	switch t := v.(type) {
	case *Enum:
		return true
	case *Typedef:
		return IsEnum(t.literal)
	}
	return false
}

func IsVoid(v interface{}) bool {
	_, ok := v.(*Void)
	return ok
}
