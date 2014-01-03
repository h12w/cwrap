// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"fmt"
	gcc "github.com/hailiang/go-gccxml"
	"reflect"
	"strings"
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

func (pac *Package) getNamer(gt gcc.Type) Namer {
	switch t := gt.(type) {
	case *gcc.FundamentalType:
		if t.CName() == "void" {
			return namer{}
		} else {
			return newNum(t)
		}
	case *gcc.Enumeration:
		return pac.newEnum(t)
	case *gcc.ArrayType:
		return pac.newArray(t)
	case *gcc.Struct:
		return pac.newStructNamer(t)
	case *gcc.Union:
		return pac.newUnionNamer(t)
	case *gcc.PointerType:
		return pac.newPtr(t.PointedType())
	case *gcc.Typedef:
		return pac.newTypedef(t)
	case *gcc.FunctionType:
		return namer{"", "[0]byte"}
	case gcc.Aliased:
		return pac.getNamer(t.Base())
	case *gcc.Unimplemented:
		return namer{}
	}
	panic(fmt.Errorf("Unkown type from gccxml: %v, %v.", reflect.TypeOf(gt), gt))
}

func (pac *Package) getConv(gt gcc.Type, ptrKind gcc.PtrKind, callback bool) Conv {
	if ptrKind == gcc.NotSet {
		n := pac.getNamer(gt)
		if named, ok := gt.(gcc.Named); ok && pac.isBool(named.CName()) {
			return Bool{n.CgoName()}
		}
		switch n.GoName() {
		case "int32", "uint32":
			return Simple{namer{"int", n.CgoName()}}
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
			return StringSlice{}
		case gcc.PtrString:
			return String{}
		case gcc.PtrTypedef:
			return pac.newPtrTypedef(gt.(*gcc.Typedef)) // Note: gt not pt.
		case gcc.PtrReference:
			if s, ok := gcc.ToComposite(pt.PointedType()); ok {
				return pac.newPtrReference(s)
			}
		case gcc.PtrReturn:
			if callback {
				return pac.newCallbackReturnPtr(pt.PointedType())
			} else {
				return pac.newReturnPtr(pt.PointedType())
			}
		}
		return pac.newPtr(pt.PointedType())
	}
	panic("Should not goes here.")
}

func (pac *Package) newReturn(gt gcc.Type) *Return {
	if gcc.IsVoid(gt) {
		return nil
	}
	var t Conv
	if gcc.IsCString(gt) {
		t = String{}
	} else if n, ok := pac.getNamer(gt).(Conv); ok {
		t = n
	}
	switch t.GoName() {
	case "int32", "uint32":
		t = Simple{namer{"int", t.CgoName()}}
	}
	return &Return{"ret", t}
}

func newNum(t *gcc.FundamentalType) Conv {
	return Simple{namer{
		goName:  goNumMap[gcc.NumInfoFromGccName(t.CName())],
		cgoName: gcc.NumCgoNameFromGccName(t.CName()),
	}}
}

func (pac *Package) newArray(t *gcc.ArrayType) Conv {
	elem := pac.getNamer(t.ElementType())
	return Value{namer{
		goName:  sprint("[", t.Len(), "]", elem.GoName()),
		cgoName: sprint("[", t.Len(), "]", elem.CgoName()),
	}}
}

func (pac *Package) newEnum(t *gcc.Enumeration) Enum {
	return Enum{
		exported: exported{
			cName: t.CName(),
			file:  t.File(),
		},
		Conv: Simple{namer{
			goName:  pac.globalName(t),
			cgoName: "C." + t.CName(),
		}},
		baseGoName: goNumMap[gcc.NumInfo{gcc.SignedInt, t.Size()}],
		Values:     pac.newEnumValues(t.EnumValues),
	}
}

func (pac *Package) newEnumValues(enumValues gcc.EnumValues) []EnumValue {
	vs := make([]EnumValue, len(enumValues))
	for i, v := range enumValues {
		vs[i] = EnumValue{pac.localName(v), v.Init()}
	}
	return vs
}

func (pac *Package) newPtrReference(t gcc.Named) Ptr {
	cgoName := "*C." + t.CName()
	if n, ok := specialCgoName(t.CName()); ok {
		cgoName = "*" + n
	}
	goName := pac.globalName(t)
	if goName == "" {
		goName = "uintptr"
	} else {
		goName = "*" + goName
	}
	return Ptr{namer{
		goName:  goName,
		cgoName: cgoName,
	}}
}

func (pac *Package) newPtrTypedef(t gcc.Named) Ptr {
	goName := pac.globalName(t)
	if goName == "" {
		goName = "uintptr"
	}
	return Ptr{namer{
		goName:  goName,
		cgoName: "C." + t.CName(),
	}}
}

func (pac *Package) newPtr(t gcc.Type) Conv {
	n := pac.getNamer(t)
	if gcc.IsVoid(t) {
		return Simple{namer{
			goName:  "uintptr",
			cgoName: "unsafe.Pointer",
		}}
	}
	goName, cgoName := n.GoName(), n.CgoName()
	if goName == "" {
		goName = "uintptr"
	} else {
		goName = "*" + goName
	}
	if cgoName == "" {
		cgoName = "unsafe.Pointer"
	} else {
		cgoName = "*" + cgoName
	}
	return Ptr{namer{
		goName:  goName,
		cgoName: cgoName,
	}}
}

func (pac *Package) newReturnPtr(t gcc.Type) ReturnPtr {
	et := pac.getNamer(t)
	return ReturnPtr{namer{
		goName:  et.GoName(),
		cgoName: "*" + et.CgoName(),
	}}
}

func (pac *Package) newCallbackReturnPtr(t gcc.Type) CallbackReturnPtr {
	et := pac.getNamer(t)
	return CallbackReturnPtr{namer{
		goName:  et.GoName(),
		cgoName: "*" + et.CgoName(),
	}}
}

func (pac *Package) newSliceSlice(t gcc.Type) SliceSlice {
	pt, _ := gcc.ToPointer(t)
	et := pac.getNamer(pt.PointedType())
	return SliceSlice{
		goName:  "[]" + et.GoName(),
		cgoName: "*" + et.CgoName(),
	}
}

func (pac *Package) newSlice(t gcc.Type) Slice {
	et := pac.getNamer(t)
	return Slice{namer{
		goName:  "[]" + et.GoName(),
		cgoName: "*" + et.CgoName(),
	}}
}

func (pac *Package) newStruct(t *gcc.Struct) Struct {
	s := pac.newStructNamer(t)
	s.Fields = pac.newStructFields(t.Fields())
	return s
}

func (pac *Package) newStructNamer(t *gcc.Struct) Struct {
	return Struct{
		exported: exported{
			cName: t.CName(),
			file:  t.File(),
		},
		Conv: Value{namer{
			goName:  pac.globalName(t),
			cgoName: "C." + t.CName(),
		}},
	}
}

func (pac *Package) newStructFields(fields gcc.Fields) []StructField {
	fs := make([]StructField, len(fields))
	for i, f := range fields {
		fs[i] = StructField{pac.upperName(f), pac.getNamer(f.CType()).GoName()}
	}
	return fs
}

func (pac *Package) newTypedef(t *gcc.Typedef) Typedef {
	baseGoName := pac.getNamer(t.Base()).GoName()
	goName := baseGoName
	if t.IsComposite() {
		goName = pac.globalName(t.Root().(gcc.Named))
	} else if n := pac.globalName(t); n != "" {
		goName = n
	}
	convFunc := convValue
	if t.IsFundamental() {
		convFunc = conv
	} else if t.IsPointer() {
		convFunc = convPtr
	}
	if t.IsComposite() || t.IsFuncType() {
		return Typedef{ // no need to define it, just name it and provide conversion.
			Namer: namer{
				goName:  goName,
				cgoName: "C." + t.CName(),
			},
			convFunc: convFunc,
		}
	}
	return Typedef{
		exported: exported{
			cName: t.CName(),
			file:  t.File(),
		},
		Namer: namer{
			goName:  goName,
			cgoName: "C." + t.CName(),
		},
		baseGoName: baseGoName,
		convFunc:   convFunc,
	}
}

func (pac *Package) newUnion(t *gcc.Union) Union {
	s := pac.newUnionNamer(t)
	s.Fields = pac.newUnionFields(t.Fields(), s.GoName())
	return s
}

func (pac *Package) newUnionNamer(t *gcc.Union) Union {
	return Union{
		exported: exported{
			cName: t.CName(),
			file:  t.File(),
		},
		Conv: Value{namer{
			goName:  pac.globalName(t),
			cgoName: "C." + t.CName(),
		}},
		baseGoName: sprint("[", t.Size()/8, "]byte"),
	}
}

func (pac *Package) newUnionFields(fields gcc.Fields, unionGoName string) []UnionField {
	fs := make([]UnionField, len(fields))
	for i, f := range fields {
		fs[i] = UnionField{pac.upperName(f), pac.getNamer(f.CType()).GoName(),
			unionGoName, uintptr(f.CType().Size() / 8)}
	}
	return fs
}

func (pac *Package) newVariable(t *gcc.Variable) Variable {
	return Variable{
		exported: exported{
			cName: t.CName(),
			file:  t.File(),
		},
		Namer: namer{
			goName:  pac.globalName(t),
			cgoName: "C." + t.CName(),
		},
		conv: pac.getNamer(t.CType()).(Conv),
	}
}

func (pac *Package) newFunction(fn *gcc.Function) Function {
	cArgs, receiver := pac.newFuncArgs(fn.Arguments)
	goName := ""
	goParams := cArgs.ToParams()
	if receiver != nil {
		goParams = goParams[1:]
		goName = pac.upperName(fn)
	} else {
		goName = pac.globalName(fn)
	}
	returns := pac.newReturn(fn.ReturnType())
	if returns != nil {
		goParams = append(goParams, returns)
	}
	return Function{
		exported: exported{
			cName: fn.CName(),
			file:  fn.File(),
		},
		goName:   goName,
		Receiver: receiver,
		GoParams: goParams,
		CArgs:    cArgs,
		Return:   returns,
	}
}

func (pac *Package) newFuncArgs(arguments gcc.Arguments) (cArgs Arguments, receiver *Receiver) {
	cArgs = pac.newArgs(arguments, false)
	if len(cArgs) > 0 &&
		arguments[0].PtrKind() == gcc.PtrReference &&
		!contains(cArgs[0].GoTypeName(), ".") &&
		cArgs[0].GoTypeName() != "uintptr" {
		objName := strings.Trim(cArgs[0].GoTypeName(), "*")
		if obj, ok := pac.structs[objName]; ok {
			receiver = &Receiver{cArgs[0], obj}
		} else if obj, ok := pac.unions[objName]; ok {
			receiver = &Receiver{cArgs[0], obj}
		} else {
			receiver = &Receiver{cArgs[0], nil}
		}
	}
	return
}

func (pac *Package) newArgs(arguments gcc.Arguments, callback bool) (args Arguments) {
	for _, a := range arguments {
		args = append(args, pac.newArg(a, callback))
	}
	return args
}

func (pac *Package) newArg(a *gcc.Argument, callback bool) Argument {
	goName := pac.lowerName(a)
	return Argument{
		namer{
			goName,
			"_" + goName,
		},
		pac.getConv(a.CType(), a.PtrKind(), callback),
		a.PtrKind() == gcc.PtrReturn,
	}
}

func (pac *Package) TransformOriginalFunc(
	oriFunc *gcc.Function,
	f CallbackFunc,
	info *gcc.CallbackInfo,
) (Function, Function) {
	fn := pac.newFunction(oriFunc)
	// GoParams
	{
		index := info.ArgIndex
		if fn.Receiver != nil {
			index--
		}
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
		fn.CArgs[info.ArgIndex] = Argument{
			namer{
				"C." + trimSuffix(f.goFuncName, "_Go") + "_C",
				ca.CgoName(),
			},
			Simple{namer{
				goName:  "",
				cgoName: ca.CgoTypeName(),
			}},
			false,
		}
		fn.CArgs[info.ArgIndex+1] = Argument{
			namer{
				"&" + ca.GoName(),
				da.CgoName(),
			},
			Simple{namer{
				goName:  "",
				cgoName: da.CgoTypeName(),
			}},
			false,
		}
	}
	fn2 := pac.newFunction(oriFunc)
	fn2.goName += "_"
	return fn, fn2
}

func (pac *Package) newCallbackFunc(info *gcc.CallbackInfo) CallbackFunc {
	callbackName := snakeToLowerCamel(upperName(info.CName, pac.From.NamePrefix)) + "Callback"
	cArgs := pac.newArgs(info.CType.Arguments, true)
	returns := pac.newReturn(info.CType.ReturnType())
	goParams := cArgs.ToParams()
	if returns != nil {
		goParams = append(goParams, returns)
	}
	return CallbackFunc{
		goFuncName:    callbackName + "_Go",
		cFuncName:     callbackName + "_C",
		GoParams:      goParams,
		CArgs:         cArgs,
		Return:        returns,
		CallbackIndex: info.DataIndex,
	}
}

func specialCgoName(n string) (string, bool) {
	switch n {
	case "__va_list_tag":
		return "_Ctype_struct___va_list_tag", true
	}
	return "", false
}
