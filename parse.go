// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"fmt"
	"io"
	"reflect"

	gcc "h12.io/go-gccxml"
)

var goNumMap = initNumMap()

// Initialize Namers in NumTypeMap with fixed size numbers.
func initNumMap() map[gcc.NumInfo]string {
	goNumMap := map[gcc.NumInfo]string{}
	goNumMap[gcc.GetNumInfo(int8(0))] = "int8"
	goNumMap[gcc.GetNumInfo(uint8(0))] = "uint8"
	goNumMap[gcc.GetNumInfo(int16(0))] = "int16"
	goNumMap[gcc.GetNumInfo(uint16(0))] = "uint16"
	goNumMap[gcc.GetNumInfo(int32(0))] = "int32"
	goNumMap[gcc.GetNumInfo(uint32(0))] = "uint32"
	goNumMap[gcc.GetNumInfo(int64(0))] = "int64"
	goNumMap[gcc.GetNumInfo(uint64(0))] = "uint64"
	goNumMap[gcc.GetNumInfo(float32(0))] = "float32"
	goNumMap[gcc.GetNumInfo(float64(0))] = "float64"
	goNumMap[gcc.GetNumInfo(complex64(0))] = "complex64"
	goNumMap[gcc.GetNumInfo(complex128(0))] = "complex128"
	goNumMap[gcc.GetNumInfo(byte(0))] = "byte" // byte overrides int8.
	// TODO: What about rune?
	return goNumMap
}

func (pac *Package) getEqualType(gt gcc.Type, declare bool) EqualType {
	if t, ok := pac.TypeDeclMap[gt.Id()]; ok {
		return t
	}
	// hard rule
	if nt, ok := gt.(gcc.Named); ok {
		if n, ok := pac.TypeRule[nt.CName()]; ok {
			return NewNum(n, cgoName(nt.CName()), gt.Size())
		}
	}
	switch t := gt.(type) {
	case *gcc.FundamentalType:
		if t.CName() == "void" {
			return &Void{}
		} else {
			return newNum(t)
		}
	case *gcc.Enumeration:
		r := newEnum(t)
		if declare {
			pac.declare(r)
		}
		return r
	case *gcc.ArrayType:
		return pac.newArray(t)
	case *gcc.Struct:
		r := newStructWithoutFields(t)
		if declare {
			pac.declare(r)
		}
		return r
	case *gcc.Union:
		r := newUnionWithoutFields(t)
		if declare {
			pac.declare(r)
		}
		return r
	case *gcc.PointerType:
		return pac.newPtr(t.PointedType())
	case *gcc.Typedef:
		r := pac.NewTypedef(t)
		if IsVoid(r.Literal) {
			return nil
		}
		if declare {
			pac.declare(r)
		}
		return r
	case *gcc.FunctionType:
		return newFuncType()
	case gcc.Aliased:
		return pac.declareEqualType(t.Base())
	case *gcc.Unimplemented:
		return nil
	}
	panic(fmt.Errorf("Unkown type from gccxml: %v, %v.", reflect.TypeOf(gt), gt))
}

func IsEnum(v interface{}) bool {
	switch t := v.(type) {
	case *Enum:
		return true
	case *Typedef:
		return IsEnum(t.Literal)
	}
	return false
}

func IsVoid(v interface{}) bool {
	return v == nil
}

func IsFunc(v interface{}) bool {
	if n, ok := v.(CgoNamer); ok {
		return n.CgoName() == "[0]byte"
	}
	return false
}

func (pac *Package) declareEqualType(gt gcc.Type) EqualType {
	return pac.getEqualType(gt, true)
}

func (pac *Package) getType(gt gcc.Type, ptrKind gcc.PtrKind) Type {
	if ptrKind == gcc.NotSet {
		if named, ok := gt.(gcc.Named); ok && pac.isBool(named.CName()) {
			return newBool(pac.declareEqualType(gt).CgoName())
		}
		return pac.declareEqualType(gt)
	}
	if pt, ok := gcc.ToPointer(gt); ok {
		pointedType := pt.PointedType()
		switch ptrKind {
		case gcc.PtrArray:
			return pac.newSlice(pointedType)
		case gcc.PtrArrayArray:
			pt, _ = gcc.ToPointer(pointedType)
			return pac.newSliceSlice(pt.PointedType())
		case gcc.PtrStringArray:
			return newStringSlice()
		case gcc.PtrString:
			return newString()
		case gcc.PtrTypedef:
			return pac.declareEqualType(gt)
		case gcc.PtrReturn:
			return pac.newReturnPtr(pointedType)
		}
		return pac.newPtr(pointedType)
	}
	panic("Should not goes here.")
}

func (pac *Package) newFunction(fn *gcc.Function) *Function {
	cArgs := pac.newArgs(fn.Arguments)
	goParams := cArgs.ToParams()
	returns := pac.newReturn(fn.ReturnType())
	if returns != nil {
		goParams = append(goParams, returns)
	}
	f := &Function{
		baseCNamer: newExported(fn),
		baseFunc: baseFunc{
			GoParams: goParams,
			CArgs:    cArgs,
			Return:   returns,
		},
	}

	return f
}

func (pac *Package) newArgs(arguments gcc.Arguments) (args Arguments) {
	for _, a := range arguments {
		args = append(args, pac.newArg(a))
	}
	return args
}

func (pac *Package) newArg(a *gcc.Argument) *Argument {
	goName := lowerName(a)
	return &Argument{
		baseParam{
			goName,
			"_" + goName,
			pac.getType(a.CType(), a.PtrKind()),
		},
		a.PtrKind() == gcc.PtrReturn,
	}
}

func (pac *Package) newReturn(gt gcc.Type) *Return {
	if gcc.IsVoid(gt) {
		return nil
	}
	var t Type
	if gcc.IsCString(gt) {
		t = newString()
	} else {
		t = pac.getType(gt, gcc.NotSet)
	}
	return &Return{baseParam{"ret", "_ret", t}}
}

func newNum(t *gcc.FundamentalType) *Num {
	return NewNum(
		goNumMap[gcc.NumInfoFromGccName(t.CName())],
		gcc.NumCgoNameFromGccName(t.CName()),
		t.Size())
}

func NewNum(goName, cgoName string, size int) *Num {
	return &Num{baseEqualType{goName, cgoName, size, NumConv}}
}

func (pac *Package) newArray(t *gcc.ArrayType) EqualType {
	return &Array{pac.declareEqualType(t.ElementType()), t.Len()}
}

func newEnum(t *gcc.Enumeration) *Enum {
	e := &Enum{
		baseEqualType: baseEqualType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
			conv:    NumConv,
		},
		baseCNamer: newExported(t),
		baseGoName: goNumMap[gcc.NumInfo{gcc.SignedInt, t.Size() * 8}],
		Values:     newEnumValues(t.EnumValues),
	}
	return e
}

func newEnumValues(enumValues gcc.EnumValues) []EnumValue {
	vs := make([]EnumValue, len(enumValues))
	for i, v := range enumValues {
		vs[i] = EnumValue{
			baseCNamer: newExported(v),
			value:      v.Init(),
		}
	}
	return vs
}

func newExported(t gcc.Named) baseCNamer {
	return baseCNamer{
		id:    t.Id(),
		cName: t.CName(),
		file:  t.File(),
	}
}

func (pac *Package) newPtr(t gcc.Type) *Ptr {
	return &Ptr{pac.declareEqualType(t)}
}

func (pac *Package) newReturnPtr(t gcc.Type) *ReturnPtr {
	return &ReturnPtr{pac.declareEqualType(t)}
}

func (pac *Package) newSliceSlice(elemType gcc.Type) *SliceSlice {
	return newSliceSlice(pac.declareEqualType(elemType))
}

func (pac *Package) newSlice(elemType gcc.Type) *Slice {
	return &Slice{elementType: pac.declareEqualType(elemType)}
}

func newStructWithoutFields(t *gcc.Struct) *Struct {
	s := &Struct{
		baseCNamer: newExported(t),
		baseEqualType: baseEqualType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
			conv:    ValConv,
		},
	}
	return s
}

func (pac *Package) newStructFields(fields gcc.Fields) []StructField {
	fs := make([]StructField, len(fields))
	for i, f := range fields {
		fs[i] = StructField{upperName(f.CName(), nil), pac.declareEqualType(f.CType())}
	}
	return fs
}

func NewSimpleTypeDef(cName, goName string, size int) *Typedef {
	return &Typedef{
		baseCNamer: baseCNamer{
			cName: cName,
		},
		baseEqualType: baseEqualType{
			cgoName: cgoName(cName),
			size:    size,
			conv:    NumConv,
		},
		Literal: NewNum(goName, "", size),
	}
}

func (pac *Package) NewTypedef(t *gcc.Typedef) *Typedef {
	var literal SpecWriter
	if t.IsEnum() {
		literal = pac.getEqualType(t.Base(), true)
	} else {
		literal = pac.getEqualType(t.Base(), false)
	}
	conv := ValConv
	if t.IsFundamental() {
		conv = NumConv
	} else if t.IsPointer() {
		conv = PtrConv
	}
	td := &Typedef{
		baseCNamer: newExported(t),
		baseEqualType: baseEqualType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
			conv:    conv,
		},
		Literal: literal,
		rootId:  t.Root().Id(),
	}
	return td
}

func newUnionWithoutFields(t *gcc.Union) *Union {
	u := &Union{
		baseCNamer: newExported(t),
		baseEqualType: baseEqualType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
			conv:    ValConv,
		},
	}
	return u
}

func (pac *Package) newUnionFields(fields gcc.Fields, union *Union) []UnionField {
	fs := make([]UnionField, len(fields))
	for i, f := range fields {
		fs[i] = UnionField{upperName(f.CName(), nil), pac.declareEqualType(f.CType()), union}
	}
	return fs
}

func (pac *Package) newVariable(t *gcc.Variable) *Variable {
	v := &Variable{
		baseCNamer: newExported(t),
		cgoName:    cgoName(t.CName()),
		conv:       pac.declareEqualType(t.CType()).(Type),
	}
	return v
}

func newBool(cgoName string) *Bool {
	return &Bool{baseType{
		"bool",
		cgoName,
	}}
}

func (pac *Package) TransformOriginalFunc(
	oriFunc *gcc.Function,
	f CallbackFunc,
	info *gcc.CallbackInfo,
) (*Function, *Function) {
	fn := pac.newFunction(oriFunc)
	// GoParams
	{
		index := info.ArgIndex
		ps, nps := fn.GoParams, Params{}
		if index > 0 {
			nps = ps[:index]
		}
		callbackArg := f.callbackArg()
		callbackArg.goName = ps[index].GoName()
		nps = append(nps, callbackArg)
		if index+1 < len(ps)-1 {
			nps = append(nps, ps[index+2:]...)
		}
		fn.GoParams = nps
	}
	// CArgs
	{
		ca, da := fn.CArgs[info.ArgIndex], fn.CArgs[info.ArgIndex+1]
		da.goName = "&" + ca.GoName()
		ca.goName = cgoName(trimSuffix(f.goName, "_Go") + "_C")
		fn.CArgs[info.ArgIndex], fn.CArgs[info.ArgIndex+1] = ca, da
		ca.isOut, da.isOut = false, false
	}
	fn2 := pac.newFunction(oriFunc)
	fn2.id += "_original"
	return fn, fn2
}

func (pac *Package) newCallbackFunc(info *gcc.CallbackInfo) CallbackFunc {
	callbackName := snakeToLowerCamel(pac.UpperName(info.CName)) + "Callback"
	cArgs := pac.newArgs(info.CType.Arguments)
	for i, a := range cArgs {
		if r, ok := a.type_.(*ReturnPtr); ok {
			cArgs[i].type_ = &CallbackReturnPtr{r}
		}
	}
	returns := pac.newReturn(info.CType.ReturnType())
	goParams := cArgs.ToParams()
	if returns != nil {
		goParams = append(goParams, returns)
	}
	return CallbackFunc{
		goName:        callbackName + "_Go",
		cFuncName:     callbackName + "_C",
		CallbackIndex: info.DataIndex,
		baseFunc: baseFunc{
			GoParams: goParams,
			CArgs:    cArgs,
			Return:   returns,
		},
		CType: info.CType,
	}
}

type CallbackReturnPtr struct {
	*ReturnPtr
}

func (s CallbackReturnPtr) ToCgo(w io.Writer, assign, g, c string) {
	conv(w, assign, g, "*"+c, s.pointedType.CgoName())
}

// lower camel name
func lowerName(o gcc.Named) string {
	s := snakeToLowerCamel(o.CName())
	switch s {
	case "func", "interface", "select", "defer", "go", "map",
		"chan", "package", "fallthrough", "range", "type", "import", "var",
		"true", "false", "iota", "nil",
		"append", "cap", "close", "complex", "copy", "delete", "imag", "len",
		"make", "new", "panic", "print", "println", "real", "recover":
		s += "_"
	}
	return s
}

func cgoName(n string) string {
	if sn, ok := specialCgoName(n); ok {
		return sn
	}
	return "C." + n
}

func generalIntFilter(n string) string {
	switch n {
	case "int32", "uint32":
	case "int64", "uint64":
		return "int"
	}
	return n
}

func specialCgoName(n string) (string, bool) {
	switch n {
	case "__va_list_tag":
		return "_Ctype_struct___va_list_tag", true
	}
	return "", false
}
