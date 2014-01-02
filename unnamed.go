// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"bytes"
	"io"
)

type Bool struct {
	cgoName string
}

func (n Bool) GoName() string {
	return "bool"
}

func (n Bool) CgoName() string {
	return n.cgoName
}

func (n Bool) ToCgo(w io.Writer, assign, g, c string) {
	fp(w, "var ", c, " ", n.CgoName())
	fp(w, "if ", g, " {")
	conv(w, "", "1", c, n.CgoName())
	fp(w, "}")
}

func (n Bool) ToGo(w io.Writer, assign, g, c string) {
	conv(w, assign, c+"==1", g, n.GoName())
}

type Slice struct {
	namer
}

func (s Slice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for Slice yet.")
}

func (s Slice) ToCgo(w io.Writer, assign, g, c string) {
	fp(w, "var ", c, " ", s.CgoName())
	fp(w, "if len(", g, ")>0 {")
	convPtr(w, "", "&"+g+"[0]", c, s.CgoName())
	fp(w, "}")
}

type SliceSlice struct {
	goName  string
	cgoName string
}

func (s SliceSlice) GoName() string {
	return "[]" + s.goName
}

func (s SliceSlice) CgoName() string {
	return "*" + s.cgoName
}

func (s SliceSlice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for SliceSlice yet.")
}

func (s SliceSlice) ToCgo(w io.Writer, assign, g, c string) {
	c_ := c + "_"
	fp(w, c_, " := make([]", s.cgoName, ", len(", g, "))")
	fp(w, "for i := range ", g, "{")
	fp(w, "if len(", g, "[i])>0 {")
	convPtr(w, "", "&"+g+"[i][0]", c_+"[i]", s.cgoName)
	fp(w, "}")
	fp(w, "}")
	Slice{namer{s.GoName(), s.CgoName()}}.ToCgo(w, assign, c_, c)
}

type StringSlice struct{}

func (s StringSlice) GoName() string {
	return "[]string"
}

func (s StringSlice) CgoName() string {
	return "**C.char"
}

func (s StringSlice) ToGo(w io.Writer, assign, g, c string) {
	fp(w, "// No ToGo conversion for StringSlice yet.")
}

func (s StringSlice) ToCgo(w io.Writer, assign, g, c string) {
	c_ := c + "_"
	fp(w, c_, " := make([]*C.char, len(", g, "))")
	fp(w, "for i := range ", g, "{")
	String{}.ToCgo(w, "", g+"[i]", c_+"[i]")
	fp(w, "}")
	Slice{namer{"[]string", "**C.char"}}.ToCgo(w, assign, c_, c)
}

type String struct{}

func (s String) GoName() string {
	return "string"
}

func (s String) CgoName() string {
	return "*C.char"
}

func (s String) ToCgo(w io.Writer, assign, g, c string) {
	fp(w, c, assign, "=C.CString(", g, ")")
	fp(w, "defer C.free(unsafe.Pointer(", c, "))")
}

func (s String) ToGo(w io.Writer, assign, g, c string) {
	fp(w, g, assign, "=C.GoString(", c, ")")
}

type ReturnPtr struct {
	namer
}

func (s ReturnPtr) ToGo(w io.Writer, assign, g, c string) {
}

func (s ReturnPtr) ToCgo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, "&"+g, c, s.CgoName())
}

type CallbackReturnPtr struct {
	namer
}

func (s CallbackReturnPtr) ToGo(w io.Writer, assign, g, c string) {
}

func (s CallbackReturnPtr) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, "*"+c, s.CgoName()[1:])
}

type CallbackFunc struct {
	goFuncName    string
	cFuncName     string
	GoParams      Params
	CArgs         Arguments
	Return        *Return
	CallbackIndex int
}

func (f CallbackFunc) Define(w io.Writer) {
	fp(w, "//export ", f.goFuncName)
	fpn(w, "func ")
	fpn(w, f.goFuncName)
	cgoParamDeclList(w, f.CArgs.ToParams()...)
	if f.Return != nil {
		cgoParamDeclList(w, f.Return)
	}
	fp(w, "{")
	f.callbackArg().ToGo(w, ":")
	for i, a := range f.CArgs {
		if i != f.CallbackIndex {
			a.ToGo(w, ":")
		}
	}
	f.internalFunc().goCall(w)
	for _, a := range f.GoParams.Out() {
		a.ToCgo(w, "")
	}
	if f.Return != nil {
		fp(w, "return")
	}
	fp(w, "}")

}

func (f CallbackFunc) internalFunc() CallbackFunc {
	return CallbackFunc{
		goFuncName: f.CArgs[f.CallbackIndex].GoName(),
		GoParams: f.GoParams.Filter(func(i int, a Param) (Param, bool) {
			return a, i != f.CallbackIndex
		}),
	}
}

func (f CallbackFunc) callbackArg() Argument {
	ca := f.CArgs[f.CallbackIndex]
	return Argument{
		namer{
			ca.GoName(),
			ca.CgoName(),
		},
		f.internalFunc(),
		false,
	}
}

func (t CallbackFunc) GoName() string {
	var buf bytes.Buffer
	t.writeGoName(&buf)
	return buf.String()
}

func (t CallbackFunc) CgoName() string {
	return "unsafe.Pointer"
}

func (t CallbackFunc) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, c, t.CgoName())
}

func (t CallbackFunc) writeGoName(w io.Writer) {
	fpn(w, "func ")
	goParamDeclListTypeOnly(w, t.GoParams.In()...)
	goParamDeclListTypeOnly(w, t.GoParams.Out()...)
}

func (t CallbackFunc) ToGo(w io.Writer, assign, g, c string) {
	fpn(w, g, assign, "=*(*")
	t.writeGoName(w)
	fp(w, ")(", c, ")")
}

func (f CallbackFunc) goCall(w io.Writer) {
	if len(f.GoParams.Out()) > 0 {
		fpn(w, f.GoParams.Out()[0].GoName())
		for _, a := range f.GoParams.Out()[1:] {
			fpn(w, ", ", a.GoName())
		}
		fpn(w, ":=")
	}
	fpn(w, f.goFuncName, "(")
	for _, a := range f.GoParams.In() {
		fpn(w, a.GoName(), ",")
	}
	fp(w, ")")
}
