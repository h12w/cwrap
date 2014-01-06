// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type FuncType struct {
	GoParams Params
	CArgs    Arguments
	Return   *Return
}

func (f *FuncType) Size() int {
	return 0
}

func (f *FuncType) GoName() string {
	return writeToString(f.WriteSpec)
}

func (f *FuncType) CgoName() string {
	return "[0]byte"
}

func (f *FuncType) WriteSpec(w io.Writer) {
	fpn(w, "func ")
	goParamDeclListTypeOnly(w, f.GoParams.In()...)
	goParamDeclListTypeOnly(w, f.GoParams.Out()...)
}

func (f *FuncType) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, c, f.CgoName())
}

func (f *FuncType) ToGo(w io.Writer, assign, g, c string) {
	fpn(w, g, assign, "=*(*")
	f.WriteSpec(w)
	fp(w, ")(", c, ")")
}

func (f *FuncType) goCall(w io.Writer, funcName string) {
	if len(f.GoParams.Out()) > 0 {
		fpn(w, f.GoParams.Out()[0].GoName())
		for _, a := range f.GoParams.Out()[1:] {
			fpn(w, ", ", a.GoName())
		}
		fpn(w, ":=")
	}
	fpn(w, funcName, "(")
	for _, a := range f.GoParams.In() {
		fpn(w, a.GoName(), ",")
	}
	fp(w, ")")
}

func (f *FuncType) cgoCall(w io.Writer, funcName string) {
	if f.Return != nil {
		fpn(w, f.Return.CgoName(), ":=")
	}
	fpn(w, "C.", funcName, "(")
	for _, a := range f.CArgs {
		fpn(w, a.CgoName(), ",")
	}
	fp(w, ")")
}

type Function struct {
	goName string
	baseCNamer
	FuncType
}

func (f *Function) GoName() string {
	return f.goName
}

func (f *Function) SetGoName(n string) {
	f.goName = n
}

func (f *Function) WriteSpec(w io.Writer) {
	f.signature(w)
	f.body(w)
}

func (f *Function) signature(w io.Writer) {
	goParamDeclList(w, f.GoParams.In()...)
	goParamDeclList(w, f.GoParams.Out()...)
}

func (f *Function) body(w io.Writer) {
	fp(w, "{")
	f.initCArgs(w)
	f.cgoCall(w, f.CName())
	f.returns(w)
	fp(w, "}")
}

func (f *Function) returns(w io.Writer) {
	if f.Return != nil {
		f.Return.ToGo(w, "")
	}
	if len(f.GoParams.Out()) > 0 {
		fp(w, "return")
	}
}

func (f *Function) initCArgs(w io.Writer) {
	for _, a := range f.CArgs {
		a.ToCgo(w, ":")
	}
}

func (f *Function) ConvertToMethod() (*Method, bool) {
	if len(f.CArgs) == 0 {
		return nil, false
	}
	recType := f.CArgs[0].GoTypeName()
	if f.CArgs[0].IsPtr() &&
		!contains(recType, ".") &&
		!contains(recType, "[") &&
		recType != "uintptr" {
		if ref, ok := f.CArgs[0].conv.(*Ptr); ok {
			if r, ok := ref.pointedType.(ReceiverType); ok {
				f.GoParams = f.GoParams[1:]
				m := &Method{f, ReceiverArg{f.CArgs[0], r}}
				m.Receiver.Type.AddMethod(m)
				return m, true
			}
		}
	}
	return nil, false
}

type Method struct {
	*Function
	Receiver ReceiverArg
}

func (m *Method) Declare(w io.Writer) {
	fp(w, "// ", m.CName())
	m.signature(w)
	m.body(w)
}

func (m *Method) signature(w io.Writer) {
	fpn(w, "func ")
	goParamDeclList(w, m.Receiver)
	fpn(w, m.GoName())
	goParamDeclList(w, m.GoParams.In()...)
	goParamDeclList(w, m.GoParams.Out()...)
}

type Methods []*Method

func (fs *Methods) AddUnique(fn *Method) {
	for _, f := range *fs {
		if f.CName() == fn.CName() {
			return
		}
	}
	fs.Append(fn)
}

func (fs *Methods) Has(goName string) bool {
	for _, f := range *fs {
		if f.GoName() == goName {
			return true
		}
	}
	return false
}

func (fs *Methods) Append(f *Method) {
	*fs = append(*fs, f)
}

type baseParam struct {
	baseType
	conv Conv
}

type Argument struct {
	baseParam
	isOut bool
}

func (a *baseParam) Conv() Conv {
	if a.conv.GoName() != "" {
		gi := generalIntFilter(a.conv.GoName())
		if gi != a.conv.GoName() {
			switch t := a.conv.(type) {
			case *ReturnPtr:
				return &ReturnPtr{newNum_("int", t.pointedType.CgoName(),
					a.conv.Size())}
			default:
				return newNum_("int", t.CgoName(), a.conv.Size())
			}
		}
	}
	return a.conv
}

func (a *Argument) IsOut() bool {
	return a.isOut
}

func (a *baseParam) ToCgo(w io.Writer, assign string) {
	a.Conv().ToCgo(w, assign, a.GoName(), a.CgoName())
}

func (a *baseParam) ToGo(w io.Writer, assign string) {
	a.Conv().ToGo(w, assign, a.GoName(), a.CgoName())
}

func (a *baseParam) CgoName() string {
	return a.cgoName
}

func (a *baseParam) GoName() string {
	return a.goName
}

func (a *baseParam) GoTypeName() string {
	return a.Conv().GoName()
}

func (a *baseParam) CgoTypeName() string {
	return a.Conv().CgoName()
}

func (a *Argument) IsPtr() bool {
	_, ok := a.Conv().(*Ptr)
	return ok
}

type Arguments []*Argument

func (as Arguments) ToParams() Params {
	ps := make(Params, len(as))
	for i, a := range as {
		ps[i] = a
	}
	return ps
}

type ReceiverType interface {
	GoName() string
	AddMethod(m *Method)
}

type ReceiverArg struct {
	*Argument
	Type ReceiverType
}

func (r *ReceiverArg) ReceiverTypeName() string {
	return r.Type.GoName()
}

type Return struct {
	baseParam
}

func (r *Return) IsOut() bool {
	return true // useless
}

type CallbackFunc struct {
	goName        string
	cFuncName     string
	CallbackIndex int
	FuncType
}

func (f CallbackFunc) Declare(w io.Writer) {
	fp(w, "//export ", f.goName)
	f.signature(w)
	f.body(w)
}

func (f CallbackFunc) signature(w io.Writer) {
	fpn(w, "func ")
	fpn(w, f.goName)
	cgoParamDeclList(w, f.CArgs.ToParams()...)
	if f.Return != nil {
		cgoParamDeclList(w, f.Return)
	}
}

func (f CallbackFunc) body(w io.Writer) {
	fp(w, "{")
	f.initGoArgs(w)
	f.internalFunc().goCall(w, f.callbackArg().GoName())
	f.returns(w)
	fp(w, "}")

}

func (f CallbackFunc) initGoArgs(w io.Writer) {
	f.callbackArg().ToGo(w, ":")
	for i, a := range f.CArgs {
		if i != f.CallbackIndex {
			a.ToGo(w, ":")
		}
	}
}

func (f CallbackFunc) returns(w io.Writer) {
	for _, a := range f.GoParams.Out() {
		a.ToCgo(w, "")
	}
	if f.Return != nil {
		fp(w, "return")
	}
}

func (f CallbackFunc) internalFunc() *FuncType {
	return &FuncType{
		GoParams: f.GoParams.Filter(func(i int, a Param) (Param, bool) {
			return a, i != f.CallbackIndex
		}),
	}
}

func (f CallbackFunc) callbackArg() *Argument {
	ca := f.CArgs[f.CallbackIndex]
	return &Argument{
		baseParam{
			baseType{
				ca.GoName(),
				ca.CgoName(),
				0,
			},
			f.internalFunc(),
		},
		false,
	}
}

func (t CallbackFunc) GoName() string {
	return writeToString(func(w io.Writer) {
		t.WriteSpec(w)
	})
}

type Param interface {
	GoName() string
	CgoName() string
	GoTypeName() string
	CgoTypeName() string
	ToCgo(w io.Writer, assign string)
	ToGo(w io.Writer, assign string)
	IsOut() bool
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
