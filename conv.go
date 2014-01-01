// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type Namer interface {
	GoName() string
	CgoName() string
}

type Conv interface {
	Namer
	ToCgo(w io.Writer, assign, g, c string)
	ToGo(w io.Writer, assign, g, c string)
}

type namer struct {
	goName  string
	cgoName string
}

func (n namer) GoName() string {
	return n.goName
}

func (n namer) CgoName() string {
	return n.cgoName
}

type Simple struct {
	Namer
}

func (n Simple) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, c, n.CgoName())
}

func (n Simple) ToGo(w io.Writer, assign, g, c string) {
	conv(w, assign, c, g, n.GoName())
}

type Ptr struct {
	Namer
}

func (t Ptr) ToCgo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, g, c, t.CgoName())
}

func (t Ptr) ToGo(w io.Writer, assign, g, c string) {
	convPtr(w, assign, c, g, t.GoName())
}

type Value struct {
	Namer
}

func (a Value) ToCgo(w io.Writer, assign, g, c string) {
	convValue(w, assign, g, c, a.CgoName())
}

func (a Value) ToGo(w io.Writer, assign, g, c string) {
	convValue(w, assign, c, g, a.GoName())
}

func conv(w io.Writer, assign, src, dst, dstType string) {
	fp(w, dst, assign, "=(", dstType, ")(", src, ")")
}

func convPtr(w io.Writer, assign, src, dst, dstType string) {
	fp(w, dst, assign, "=(", dstType, ")(unsafe.Pointer(", src, "))")
}

func convValue(w io.Writer, assign, src, dst, dstType string) {
	fp(w, dst, assign, "=*(*", dstType, ")(unsafe.Pointer(&", src, "))")
}
