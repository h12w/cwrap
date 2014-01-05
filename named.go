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
	conv    Conv
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
	v.conv.ToGo(w, "", v.GoName(), v.CgoName())
}

type Enum struct {
	baseCNamer
	baseType
	SimpleConv
	baseGoName string
	Values     []EnumValue
}

func (*Enum) WriteMethods(io.Writer) {
}

func (e *Enum) WriteSpec(w io.Writer) {
	fp(w, e.baseGoName)
	fp(w, "const (")
	for _, v := range e.Values {
		v.Declare(w)
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

func (v *EnumValue) Declare(w io.Writer) {
	fp(w, v.goName, "=", v.value)
}

type Typedef struct {
	baseCNamer
	baseType
	literal  SpecWriter
	convFunc func(io.Writer, string, string, string, string)
	receiver
	rootId string
}

func (d *Typedef) GoName() string {
	if d.goName == "" {
		return d.Root().GoName()
	}
	return d.goName
}

func (d *Typedef) Root() Type {
	switch t := d.literal.(type) {
	case *Typedef:
		return t.Root()
	}
	return d.literal.(Type)
}

func (d *Typedef) OptimizeNames() {
	d.receiver.OptimizeNames(d.GoName())
	if o, ok := d.literal.(NameOptimizer); ok {
		o.OptimizeNames()
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
	d.receiver.WriteMethods(w)
}

func (d *Typedef) ToCgo(w io.Writer, assign, g, c string) {
	d.convFunc(w, assign, g, c, d.CgoName())
}

func (d *Typedef) ToGo(w io.Writer, assign, g, c string) {
	d.convFunc(w, assign, c, g, d.GoName())
}

type receiver struct {
	Methods
}

func (s *receiver) AddMethod(m *Method) {
	s.Methods.AddUnique(m)
}

func (s *receiver) WriteMethods(w io.Writer) {
	for _, m := range s.Methods {
		m.Declare(w)
	}
}

func (s *receiver) OptimizeNames(typeName string) {
	for i, m := range s.Methods {
		newName := trimPreSuffix(m.GoName(), typeName)
		if newName != "" && !s.Methods.Has(newName) {
			s.Methods[i].SetGoName(newName)
		}
	}
}

type Struct struct {
	baseCNamer
	baseType
	ValueConv
	Fields []StructField
	receiver
}

func (s *Struct) OptimizeNames() {
	for i, f := range s.Fields {
		if s.Methods.Has(f.goName) {
			s.Fields[i].goName += "_"
		}
	}
	s.receiver.OptimizeNames(s.GoName())
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
	Type   GoNamer
}

func (f *StructField) Declare(w io.Writer) {
	fp(w, f.goName, " ", f.Type.GoName())
}

type Union struct {
	baseCNamer
	baseType
	ValueConv
	size   int
	Fields []UnionField
	receiver
}

func (s *Union) WriteMethods(w io.Writer) {
	for _, f := range s.Fields {
		f.Declare(w)
	}
	s.receiver.WriteMethods(w)
}

func (s *Union) WriteSpec(w io.Writer) {
	fp(w, "[", s.size, "]byte")
}

type UnionField struct {
	goName string
	Type   GoNamer
	size   uintptr
	union  *Union
}

func (f *UnionField) Declare(w io.Writer) {
	if f.size <= MachineSize {
		f.defineValueGetter(w)
	} else {
		f.definePtrGetter(w)
	}
}

func (f *UnionField) defineValueGetter(w io.Writer) {
	fp(w, "func (u *", f.union.GoName(), ")", f.goName, "() ",
		f.Type.GoName(), "{")
	fp(w, "return ", "*(*", f.Type.GoName(), ")(unsafe.Pointer(u))")
	fp(w, "}")
}

func (f *UnionField) definePtrGetter(w io.Writer) {
	fp(w, "func (u *", f.union.GoName(), ")", f.goName, "() *",
		f.Type.GoName(), "{")
	fp(w, "return ", "(*", f.Type.GoName(), ")(unsafe.Pointer(u))")
	fp(w, "}")
}
