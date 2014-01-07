// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
)

type Variable struct {
	baseCNamer
	goName  string
	cgoName string
	conv    TypeConv
}

func (v *Variable) GoName() string {
	return v.goName
}

func (v *Variable) SetGoName(n string) {
	v.goName = n
}

func (v *Variable) CgoName() string {
	return v.cgoName
}

func (v *Variable) WriteSpec(w io.Writer) {
	v.conv.ToGo(w, "", "", v.CgoName())
}

type Enum struct {
	baseCNamer
	baseEqualType
	baseGoName string
	Values     []EnumValue
	Methods
}

func (e *Enum) GoName() string {
	return e.goName
}

func (e *Enum) WriteSpec(w io.Writer) {
	if e.GoName() != "" {
		fp(w, e.baseGoName)
	}
	fp(w, "const (")
	length := 0
	for _, v := range e.Values {
		l := len(hex(v.value, 0))
		if length < l {
			length = l
		}
	}
	for _, v := range e.Values {
		fp(w, v.goName, "=", hex(v.value, length))
	}
	fp(w, ")")
}

type EnumValue struct {
	baseCNamer
	goName string
	value  int
}

func (v *EnumValue) GoName() string {
	return v.goName
}

type Typedef struct {
	baseCNamer
	baseEqualType
	literal SpecWriter
	Methods
	rootId string
}

func (d *Typedef) GoName() string {
	if d.goName == "" {
		d.goName = d.Root().GoName()
	}
	return d.goName
}

func (d *Typedef) Root() EqualType {
	switch t := d.literal.(type) {
	case *Typedef:
		return t.Root()
	}
	return d.literal.(EqualType)
}

func (d *Typedef) OptimizeNames() {
	d.Methods.OptimizeNames(d.GoName())
	if o, ok := d.literal.(NameOptimizer); ok {
		o.OptimizeNames()
	}
	if s, ok := d.literal.(*Struct); ok {
		s.OptimizeFieldNames(d.Methods)
	}
}

func (d *Typedef) WriteSpec(w io.Writer) {
	d.literal.WriteSpec(w)
}

func (d *Typedef) WriteMethods(w io.Writer) {
	if u, ok := d.literal.(*Union); ok {
		goName := u.GoName()
		u.SetGoName(d.GoName())
		u.WriteMethods(w)
		u.SetGoName(goName)
	}
	d.Methods.WriteMethods(w)
}

type Struct struct {
	baseCNamer
	baseEqualType
	Fields []StructField
	Methods
}

func (s *Struct) OptimizeNames() {
	s.OptimizeFieldNames(s.Methods)
	s.Methods.OptimizeNames(s.GoName())
}

func (s *Struct) OptimizeFieldNames(methods Methods) {
	for i, f := range s.Fields {
		if methods.Has(f.goName) {
			s.Fields[i].goName += "_"
		}
	}
}

func (s *Struct) WriteSpec(w io.Writer) {
	fp(w, "struct {")
	for _, f := range s.Fields {
		f.Declare(w)
	}
	fp(w, "}")
}

type StructField struct {
	goName string
	EqualType
}

func (f *StructField) Declare(w io.Writer) {
	fp(w, f.goName, " ", f.EqualType.GoName())
}

type Union struct {
	baseCNamer
	baseEqualType
	Fields []UnionField
	Methods
}

func (s *Union) OptimizeNames() {
	s.Methods.OptimizeNames(s.GoName())
}

func (s *Union) WriteMethods(w io.Writer) {
	for _, f := range s.Fields {
		f.Declare(w)
	}
	s.Methods.WriteMethods(w)
}

func (s *Union) WriteSpec(w io.Writer) {
	fp(w, "[", s.size, "]byte")
}

type UnionField struct {
	goName string
	EqualType
	union *Union
}

func (f *UnionField) Declare(w io.Writer) {
	if f.Size() <= MachineSize {
		f.defineValueGetter(w)
	} else {
		f.definePtrGetter(w)
	}
}

func (f *UnionField) defineValueGetter(w io.Writer) {
	fp(w, "func (u *", f.union.GoName(), ")", f.goName, "() ",
		f.EqualType.GoName(), "{")
	fp(w, "return ", "*(*", f.EqualType.GoName(), ")(unsafe.Pointer(u))")
	fp(w, "}")
}

func (f *UnionField) definePtrGetter(w io.Writer) {
	fp(w, "func (u *", f.union.GoName(), ")", f.goName, "() *",
		f.EqualType.GoName(), "{")
	fp(w, "return ", "(*", f.EqualType.GoName(), ")(unsafe.Pointer(u))")
	fp(w, "}")
}
