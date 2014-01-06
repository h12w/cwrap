// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"fmt"
	gcc "github.com/hailiang/go-gccxml"
	"io"
	"reflect"
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

func (pac *Package) getType(gt gcc.Type, declare bool) Type {
	if t, ok := pac.typeDeclMap[gt.Id()]; ok {
		return t
	}
	// hard rule
	if nt, ok := gt.(gcc.Named); ok {
		if n, ok := pac.TypeRule[nt.CName()]; ok {
			return newNum_(n, cgoName(nt.CName()), gt.Size())
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
		r := pac.newTypedef(t)
		if IsVoid(r.literal) {
			return &Void{}
		}
		if declare {
			pac.declare(r)
		}
		return r
	case *gcc.FunctionType:
		return &baseType{"[0]byte", "[0]byte", 0}
	case gcc.Aliased:
		return pac.declareType(t.Base())
	case *gcc.Unimplemented:
		return &baseType{}
	}
	panic(fmt.Errorf("Unkown type from gccxml: %v, %v.", reflect.TypeOf(gt), gt))
}

func (pac *Package) declareType(gt gcc.Type) Type {
	return pac.getType(gt, true)
}

func (pac *Package) getConv(gt gcc.Type, ptrKind gcc.PtrKind) Conv {
	if ptrKind == gcc.NotSet {
		n := pac.declareType(gt)
		if named, ok := gt.(gcc.Named); ok && pac.isBool(named.CName()) {
			return newBool(n.CgoName(), gt.Size())
		}
		if argType, ok := n.(Conv); ok {
			return argType
		}
	}
	if pt, ok := gcc.ToPointer(gt); ok {
		switch ptrKind {
		case gcc.PtrArray:
			return pac.newSlice(pt.PointedType())
		case gcc.PtrArrayArray:
			return pac.newSliceSlice(pt.PointedType())
		case gcc.PtrStringArray:
			return newStringSlice()
		case gcc.PtrString:
			return newString()
		case gcc.PtrTypedef:
			return pac.declareType(gt).(Conv)
		case gcc.PtrReturn:
			return pac.newReturnPtr(pt.PointedType())
		}
		return pac.newPtr(pt.PointedType())
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
		FuncType: FuncType{
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
			baseType{
				goName,
				"_" + goName,
				a.CType().Size(),
			},
			pac.getConv(a.CType(), a.PtrKind()),
		},
		a.PtrKind() == gcc.PtrReturn,
	}
}

func (pac *Package) newReturn(gt gcc.Type) *Return {
	if gcc.IsVoid(gt) {
		return nil
	}
	var t Conv
	if gcc.IsCString(gt) {
		t = newString()
	} else {
		t = pac.getConv(gt, gcc.NotSet)
	}
	return &Return{baseParam{baseType{"ret", "_ret", gt.Size()}, t}}
}

func newNum(t *gcc.FundamentalType) *Num {
	return &Num{SimpleConv{&baseType{
		goName:  goNumMap[gcc.NumInfoFromGccName(t.CName())],
		cgoName: gcc.NumCgoNameFromGccName(t.CName()),
		size:    t.Size(),
	}}}
}

func newNum_(goName, cgoName string, size int) *Num {
	return &Num{SimpleConv{&baseType{goName, cgoName, size}}}
}

func (pac *Package) newArray(t *gcc.ArrayType) Conv {
	return ValueConv{&Array{pac.declareType(t.ElementType()), t.Len()}}
}

func newEnum(t *gcc.Enumeration) *Enum {
	e := &Enum{
		baseType: baseType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
		},
		baseCNamer: newExported(t),
		baseGoName: goNumMap[gcc.NumInfo{gcc.SignedInt, t.Size() * 8}],
		Values:     newEnumValues(t.EnumValues),
	}
	e.SimpleConv = SimpleConv{e}
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
	return &Ptr{pac.declareType(t)}
}

func (pac *Package) newReturnPtr(t gcc.Type) *ReturnPtr {
	return &ReturnPtr{pac.declareType(t)}
}

func (pac *Package) newSliceSlice(t gcc.Type) *SliceSlice {
	pt, _ := gcc.ToPointer(t)
	return &SliceSlice{Slice{pac.declareType(pt.PointedType())}}
}

func (pac *Package) newSlice(t gcc.Type) *Slice {
	return &Slice{pac.declareType(t)}
}

func newStructWithoutFields(t *gcc.Struct) *Struct {
	s := &Struct{
		baseCNamer: newExported(t),
		baseType: baseType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
		},
	}
	s.ValueConv = ValueConv{s}
	return s
}

func (pac *Package) newStructFields(fields gcc.Fields) []StructField {
	fs := make([]StructField, len(fields))
	for i, f := range fields {
		fs[i] = StructField{upperName(f.CName(), nil), pac.declareType(f.CType())}
	}
	return fs
}

func (pac *Package) newTypedef(t *gcc.Typedef) *Typedef {
	var literal SpecWriter
	if t.IsEnum() {
		literal = pac.declareType(t.Base())
	} else {
		literal = pac.getType(t.Base(), false)
	}
	convFunc := convValue
	if t.IsFundamental() {
		convFunc = conv
	} else if t.IsPointer() {
		convFunc = convPtr
	}
	td := &Typedef{
		baseCNamer: newExported(t),
		baseType: baseType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
		},
		literal:  literal,
		convFunc: convFunc,
		rootId:   t.Root().Id(),
	}
	return td
}

func newUnionWithoutFields(t *gcc.Union) *Union {
	u := &Union{
		baseCNamer: newExported(t),
		baseType: baseType{
			cgoName: cgoName(t.CName()),
			size:    t.Size(),
		},
		size: t.Size(),
	}
	u.ValueConv = ValueConv{u}
	return u
}

func (pac *Package) newUnionFields(fields gcc.Fields, union *Union) []UnionField {
	fs := make([]UnionField, len(fields))
	for i, f := range fields {
		fs[i] = UnionField{upperName(f.CName(), nil), pac.declareType(f.CType()),
			f.CType().Size(), union}
	}
	return fs
}

func (pac *Package) newVariable(t *gcc.Variable) *Variable {
	v := &Variable{
		baseCNamer: newExported(t),
		cgoName:    cgoName(t.CName()),
		conv:       pac.declareType(t.CType()).(Conv),
	}
	return v
}

func newBool(cgoName string, size int) *Bool {
	return &Bool{baseType{
		"bool",
		cgoName,
		size,
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
	callbackName := snakeToLowerCamel(pac.upperName(info.CName)) + "Callback"
	cArgs := pac.newArgs(info.CType.Arguments)
	for i, a := range cArgs {
		if r, ok := a.conv.(*ReturnPtr); ok {
			cArgs[i].conv = &CallbackReturnPtr{r}
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
		FuncType: FuncType{
			GoParams: goParams,
			CArgs:    cArgs,
			Return:   returns,
		},
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
