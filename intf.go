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

type TypeConv interface {
	ToCgo(w io.Writer, assign, g, c string)
	ToGo(w io.Writer, assign, g, c string)
}

type NameOptimizer interface {
	OptimizeNames()
}

type Type interface {
	GoNamer
	CgoNamer
	TypeConv
}

type EqualType interface {
	Type
	Size() int
	SpecWriter
}

type ReceiverType interface {
	EqualType
	AddMethod(m *Method)
	MethodsWriter
}

type Decl interface {
	CNamer
	GoNamer
	GoNameSetter
	SpecWriter
}

type TypeDecl interface {
	ReceiverType
	CNamer
	GoNameSetter
}

type baseType struct {
	goName  string
	cgoName string
}

func (t *baseType) SetGoName(n string) {
	t.goName = n
}

func (t *baseType) GoName() string {
	return t.goName
}

func (t *baseType) CgoName() string {
	return t.cgoName
}

type baseEqualType struct {
	goName  string
	cgoName string
	size    int
	conv    Conv
}

func (t *baseEqualType) Size() int {
	return t.size
}

func (t *baseEqualType) SetGoName(n string) {
	t.goName = n
}

func (t *baseEqualType) GoName() string {
	if t.goName == "" {
		t.goName = sprint("[", t.size, "]byte")
	}
	return t.goName
}

func (t *baseEqualType) CgoName() string {
	return t.cgoName
}

func (t *baseEqualType) WriteSpec(w io.Writer) {
	fpn(w, t.goName)
}

func (t *baseEqualType) ToCgo(w io.Writer, assign, g, c string) {
	t.conv.ToCgo(w, assign, g, c, t.cgoName)
}

func (t *baseEqualType) ToGo(w io.Writer, assign, g, c string) {
	t.conv.ToGo(w, assign, g, c, t.goName)
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

type Conv interface {
	ToCgo(w io.Writer, assign, g, c, ctype string)
	ToGo(w io.Writer, assign, g, c, gtype string)
}

type ConvFunc func(io.Writer, string, string, string, string)

type convImpl struct {
	toCgo ConvFunc
	toGo  ConvFunc
}

func (conv *convImpl) ToCgo(w io.Writer, assign, g, c, ctype string) {
	conv.toCgo(w, assign, g, c, ctype)
}

func (conv *convImpl) ToGo(w io.Writer, assign, g, c, gtype string) {
	conv.toGo(w, assign, c, g, gtype)
}

var (
	NumConv = &convImpl{conv, conv}
	PtrConv = &convImpl{convPtr, convPtr}
	ValConv = &convImpl{convValue, convValue}
)

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
