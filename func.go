// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"

	"h12.io/go-gccxml"
)

type baseFunc struct {
	GoParams Params
	CArgs    Arguments
	Return   *Return
}

func (f *baseFunc) GoName() string {
	return writeToString(f.WriteSpec)
}

func (f *baseFunc) CgoName() string {
	return "[0]byte"
}

func (f *baseFunc) WriteSpec(w io.Writer) {
	fpn(w, "func ")
	goParamDeclListTypeOnly(w, f.GoParams.In()...)
	goParamDeclListTypeOnly(w, f.GoParams.Out()...)
}

func (f *baseFunc) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, c, f.CgoName())
}

func (f *baseFunc) ToGo(w io.Writer, assign, g, c string) {
	fpn(w, g, assign, "=*(*")
	f.WriteSpec(w)
	fp(w, ")(", c, ")")
}

func (f *baseFunc) goCall(w io.Writer, funcName string) {
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

func (f *baseFunc) cgoCall(w io.Writer, funcName string) {
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
	baseFunc
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
	first := f.CArgs[0]
	recType := first.GoTypeName()
	if first.IsPtr() &&
		!contains(recType, ".") &&
		!contains(recType, "[") &&
		recType != "uintptr" {
		if ref, ok := first.type_.(*Ptr); ok {
			if r, ok := ref.pointedType.(ReceiverType); ok {
				f.GoParams = f.GoParams[1:]
				m := &Method{f, ReceiverArg{Argument: first, EqualType: r}}
				r.AddMethod(m)
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

func (ms *Methods) Has(methodName string) bool {
	for _, m := range *ms {
		if m.GoName() == methodName {
			return true
		}
	}
	return false
}

func (ms *Methods) AddMethod(method *Method) {
	for _, m := range *ms {
		if m.CName() == method.CName() {
			return
		}
	}
	*ms = append(*ms, method)
}

func (ms *Methods) WriteMethods(w io.Writer) {
	for _, m := range *ms {
		m.Declare(w)
	}
}

func (ms *Methods) OptimizeNames(typeName string) {
	for i, m := range *ms {
		newName := replace(m.GoName(), typeName, "")
		if newName != "" && !ms.Has(newName) {
			(*ms)[i].SetGoName(newName)
		}
	}
}

type baseParam struct {
	goName  string
	cgoName string
	type_   Type
}

type Argument struct {
	baseParam
	isOut bool
}

func NewArgument(goName, cgoName string, typ Type) *Argument {
	return &Argument{
		baseParam: baseParam{
			goName:  goName,
			cgoName: cgoName,
			type_:   typ,
		},
	}
}

func (a *baseParam) Type() Type {
	// this logic cannot be put into getType because it must wait till all type's
	// GoNames are settled.
	if a.type_.GoName() != "" {
		gi := generalIntFilter(a.type_.GoName())
		if gi != a.type_.GoName() {
			switch t := a.type_.(type) {
			case *ReturnPtr:
				return &ReturnPtr{NewNum("int", t.pointedType.CgoName(),
					MachineSize)}
			default:
				return NewNum("int", t.CgoName(), MachineSize)
			}
		}
	}
	return a.type_
}

func (a *Argument) IsOut() bool {
	return a.isOut
}

func (a *baseParam) ToCgo(w io.Writer, assign string) {
	a.Type().ToCgo(w, assign, a.GoName(), a.CgoName())
}

func (a *baseParam) ToGo(w io.Writer, assign string) {
	a.Type().ToGo(w, assign, a.GoName(), a.CgoName())
}

func (a *baseParam) CgoName() string {
	return a.cgoName
}

func (a *baseParam) GoName() string {
	return a.goName
}

func (a *baseParam) GoTypeName() string {
	return a.Type().GoName()
}

func (a *baseParam) CgoTypeName() string {
	return a.Type().CgoName()
}

func (a *Argument) IsPtr() bool {
	_, ok := a.Type().(*Ptr)
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

type ReceiverArg struct {
	*Argument
	EqualType ReceiverType
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
	baseFunc
	CType *gccxml.FunctionType
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

func (f CallbackFunc) internalFunc() *baseFunc {
	return &baseFunc{
		GoParams: f.GoParams.Filter(func(i int, a Param) (Param, bool) {
			return a, i != f.CallbackIndex
		}),
	}
}

func (f CallbackFunc) callbackArg() *Argument {
	ca := f.CArgs[f.CallbackIndex]
	return &Argument{
		baseParam{
			ca.GoName(),
			ca.CgoName(),
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
		cgoType := d.CgoTypeName()
		if cgoType != "*C.char" && hasPrefix(cgoType, "*") {
			cgoType = "unsafe.Pointer"
		}
		fpn(w, d.CgoName(), " ", cgoType, ",")
	}
	fpn(w, ")")
}
