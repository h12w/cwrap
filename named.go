// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type NamedType interface {
	CName() string
	File() string
	Define(io.Writer)
}

type exported struct {
	cName string
	file  string
}

func (e exported) CName() string {
	return e.cName
}

func (e exported) File() string {
	return e.file
}

type Variable struct {
	exported
	Namer
	conv Conv
}

func (v Variable) Define(w io.Writer) {
	fpn(w, "var ")
	v.conv.ToGo(w, "", v.GoName(), v.CgoName())
}

type Enum struct {
	exported
	Conv
	baseGoName string
	Values     []EnumValue
}

func (e Enum) Define(w io.Writer) {
	fp(w, "type ", e.GoName(), " ", e.baseGoName)
	fp(w, "const (")
	for _, v := range e.Values {
		v.Define(w)
	}
	fp(w, ")")
}

type EnumValue struct {
	GoName string
	Value  int
}

func (v EnumValue) Define(w io.Writer) {
	fp(w, v.GoName, "=", v.Value)
}

type Typedef struct {
	exported
	Namer
	baseGoName string
	convFunc   func(io.Writer, string, string, string, string)
}

func (d Typedef) Define(w io.Writer) {
	fp(w, "type ", d.GoName(), " ", d.baseGoName)
}

func (d Typedef) isValid() bool {
	return d.baseGoName != ""
}

func (d Typedef) ToCgo(w io.Writer, assign, g, c string) {
	d.convFunc(w, assign, g, c, d.CgoName())
}

func (d Typedef) ToGo(w io.Writer, assign, g, c string) {
	d.convFunc(w, assign, c, g, d.GoName())
}

type Struct struct {
	exported
	Conv
	Fields []StructField
}

type StructField struct {
	GoName     string
	GoTypeName string
}

func (f StructField) Define(w io.Writer) {
	fp(w, f.GoName, " ", f.GoTypeName)
}

func (s Struct) Define(w io.Writer) {
	fp(w, "type ", s.GoName(), " struct {")
	for _, f := range s.Fields {
		f.Define(w)
	}
	fp(w, "}")
}

type Union struct {
	exported
	Conv
	baseGoName string
	Fields     []UnionField
}

func (s Union) Define(w io.Writer) {
	fp(w, "type ", s.GoName(), " ", s.baseGoName)
	for _, f := range s.Fields {
		f.Define(w)
	}
}

type UnionField struct {
	goName      string
	goTypeName  string
	unionGoName string
	size        uintptr
}

func (f UnionField) Define(w io.Writer) {
	if f.size <= MachineSize {
		f.defineValueGetter(w)
	} else {
		f.definePtrGetter(w)
	}
}

func (f UnionField) defineValueGetter(w io.Writer) {
	fp(w, "func (u *", f.unionGoName, ")", f.goName, "() ",
		f.goTypeName, "{")
	fp(w, "return ", "*(*", f.goTypeName, ")(unsafe.Pointer(u))")
	fp(w, "}")
}

func (f UnionField) definePtrGetter(w io.Writer) {
	fp(w, "func (u *", f.unionGoName, ")", f.goName, "() *",
		f.goTypeName, "{")
	fp(w, "return ", "(*", f.goTypeName, ")(unsafe.Pointer(u))")
	fp(w, "}")
}

type Param interface {
	GoName() string
	CgoName() string
	GoTypeName() string
	CgoTypeName() string
	IsOut() bool
	ToCgo(w io.Writer, assign string)
	ToGo(w io.Writer, assign string)
}

type Params []Param

func (ps Params) Filter(filter func(i int, a Param) (Param, bool)) (as Params) {
	for i, a := range ps {
		if fa, ok := filter(i, a); ok {
			as = append(as, fa)
		}
	}
	return
}

type Function struct {
	exported
	goName   string
	Receiver Param
	GoParams Params
	CArgs    []Argument
	Return   *Return
}

func (f Function) Define(w io.Writer) {
	f.signature(w)
	f.body(w)
}

func (f Function) signature(w io.Writer) {
	fpn(w, "func ")
	if f.Receiver != nil {
		goParamDeclList(w, f.Receiver)
	}
	fpn(w, f.goName)
	goParamDeclList(w, f.GoParams.In()...)
	goParamDeclList(w, f.GoParams.Out()...)
}

func (f Function) body(w io.Writer) {
	fp(w, "{")
	f.initCArgs(w)
	f.cgoCall(w)
	f.returns(w)
	fp(w, "}")
}

func (f Function) returns(w io.Writer) {
	if f.Return != nil {
		f.Return.ToGo(w, "")
	}
	if len(f.GoParams.Out()) > 0 {
		fp(w, "return")
	}
}

func (ps Params) In() Params {
	return ps.Filter(func(i int, a Param) (Param, bool) {
		if a.IsOut() {
			return a, false
		}
		return a, true
	})
}

func (ps Params) Out() Params {
	return ps.Filter(func(i int, a Param) (Param, bool) {
		if a.IsOut() {
			return a, true
		}
		return a, false
	})
}

func goParamDeclList(w io.Writer, ds ...Param) {
	fpn(w, "(")
	for _, d := range ds {
		fpn(w, d.GoName(), " ", d.GoTypeName(), ",")
	}
	fpn(w, ")")
}

func goParamDeclListTypeOnly(w io.Writer, ds ...Param) {
	fpn(w, "(")
	for _, d := range ds {
		fpn(w, d.GoTypeName(), ",")
	}
	fpn(w, ")")
}

func cgoParamDeclList(w io.Writer, ds ...Param) {
	fpn(w, "(")
	for _, d := range ds {
		fpn(w, d.CgoName(), " ", d.CgoTypeName(), ",")
	}
	fpn(w, ")")
}

func (f Function) initCArgs(w io.Writer) {
	for _, a := range f.CArgs {
		a.ToCgo(w, ":")
	}
}

func (f Function) cgoCall(w io.Writer) {
	if f.Return != nil {
		fpn(w, f.Return.CgoName(), ":=")
	}
	fpn(w, "C.", f.CName(), "(")
	for _, a := range f.CArgs {
		fpn(w, a.CgoName(), ",")
	}
	fp(w, ")")
}

type Argument struct {
	namer
	conv  Conv
	isOut bool
}

type Arguments []Argument

func (as Arguments) ToParams() Params {
	ps := make(Params, len(as))
	for i, a := range as {
		ps[i] = a
	}
	return ps
}

func (a Argument) IsOut() bool {
	return a.isOut
}

func (a Argument) ToCgo(w io.Writer, assign string) {
	a.conv.ToCgo(w, assign, a.GoName(), a.CgoName())
}

func (a Argument) ToGo(w io.Writer, assign string) {
	a.conv.ToGo(w, assign, a.GoName(), a.CgoName())
}

func (a Argument) CgoName() string {
	return a.cgoName
}

func (a Argument) GoName() string {
	return a.goName
}

func (a Argument) GoTypeName() string {
	return a.conv.GoName()
}

func (a Argument) CgoTypeName() string {
	return a.conv.CgoName()
}

func (a Argument) IsReference() bool {
	_, ok := a.conv.(Ptr)
	return ok
}

type Return struct {
	goName string
	conv   Conv
}

func (f Return) IsOut() bool {
	return true // useless
}

func (r Return) ToGo(w io.Writer, assign string) {
	r.conv.ToGo(w, assign, r.GoName(), r.CgoName())
}

func (r Return) ToCgo(w io.Writer, assign string) {
	r.conv.ToCgo(w, assign, r.GoName(), r.CgoName())
}

func (r Return) GoName() string {
	return r.goName
}

func (r Return) CgoName() string {
	return "_" + r.GoName()
}

func (r Return) GoTypeName() string {
	return r.conv.GoName()
}

func (r Return) CgoTypeName() string {
	return r.conv.CgoName()
}
