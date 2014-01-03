// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	gcc "github.com/hailiang/go-gccxml"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"sort"
	"syscall"
)

var (
	GOPATH, _ = syscall.Getenv("GOPATH")
	OutputDir = GOPATH + "/src/"
)

type Header struct {
	Dir        string
	File       string
	NamePrefix string
	OtherCode  string
	// Not define it in the package, but may still searchable as included types
	// because it may be manually defined.
	Excluded      []string
	CgoDirectives []string
	BoolTypes     []string
}

func (h Header) FullPath() string {
	file := h.Dir + h.File
	if !fileExists(file) {
		panic("Header file cannot be found: " + file)
	}
	return file
}

func (h Header) Write(w io.Writer) {
	fp(w, h.OtherCode)
	fp(w, "#include <", h.File, ">")
}

type Package struct {
	// Required
	PacName string
	PacPath string
	From    Header

	// Optional
	Included []*Package
	GoFile   string
	CFile    string
	HFile    string

	// Internal
	localNames map[string]string
	fileIds    SSet
	boolSet    SSet
	structs    map[string]*Struct // Key: go name
	unions     map[string]*Union  // Key: go name
	functions  Functions
	callbacks  []CallbackFunc
	Statistics
	*gcc.XmlDoc
}

func (pac *Package) Load() (err error) {
	pac.localNames = make(map[string]string)
	pac.initBoolSet()
	pac.structs = make(map[string]*Struct)
	pac.unions = make(map[string]*Union)
	if err := pac.loadXmlDoc(); err != nil {
		return err
	}
	if err := pac.initFileIds(); err != nil {
		return err
	}
	for _, inc := range pac.Included {
		inc.XmlDoc = pac.XmlDoc
		if err := inc.Load(); err != nil {
			return err
		}
	}
	return nil
}

func (pac *Package) loadXmlDoc() error {
	if pac.XmlDoc != nil {
		return nil
	}
	f, err := ioutil.TempFile(".", "_cwrap-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	for _, inc := range pac.Included {
		inc.From.Write(f)
	}
	pac.From.Write(f)
	f.Close()
	pac.XmlDoc, err = gcc.Xml{f.Name()}.Doc()
	//	pac.XmlDoc.Print()
	return err
}

func (pac *Package) initBoolSet() {
	pac.boolSet = NewSSet()
	for _, t := range pac.From.BoolTypes {
		pac.boolSet.Add(t)
	}
}

func (pac *Package) initFileIds() error {
	pac.fileIds = NewSSet()
	fnames, err := gcc.IncludeFiles(pac.From.FullPath())
	if err != nil {
		return err
	}
	// if includedFiles failed, at least has one file( builtin)
	fnames = append(fnames, pac.From.FullPath())
	for _, name := range fnames {
		for _, file := range pac.XmlDoc.Files {
			if file.CName() == name {
				pac.fileIds.Add(file.Id())
				break
			}
		}
	}
	return nil
}

func (pac *Package) goFile() string {
	if pac.GoFile != "" {
		return pac.GoFile
	}
	return pac.defaultFile() + ".go"
}

func (pac *Package) cFile() string {
	if pac.CFile != "" {
		return pac.CFile
	}
	return pac.defaultFile() + ".c"
}

func (pac *Package) hFile() string {
	if pac.HFile != "" {
		return pac.HFile
	}
	return pac.defaultFile() + ".h"
}

func (pac *Package) defaultFile() string {
	return OutputDir + pac.PacPath + "/auto_" + runtime.GOARCH
}

func (pac *Package) createFile(file string) (io.WriteCloser, error) {
	if err := os.MkdirAll(path.Dir(file), 0755); err != nil {
		return nil, err
	}
	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (pac *Package) Wrap() error {
	g, err := pac.createFile(pac.goFile())
	if err != nil {
		return err
	}
	defer g.Close()
	c, err := pac.createFile(pac.cFile())
	if err != nil {
		return err
	}
	defer c.Close()
	h, err := pac.createFile(pac.hFile())
	if err != nil {
		return err
	}
	defer h.Close()
	if err := pac.prepare(); err != nil {
		return err
	}
	if err := pac.write(g, c, h); err != nil {
		return err
	}
	return gofmt(pac.goFile())
}

func (pac *Package) prepare() error {
	if pac.XmlDoc == nil {
		if err := pac.Load(); err != nil {
			return err
		}
	}
	for _, st := range pac.Structs {
		s := pac.newStruct(st)
		pac.structs[s.GoName()] = &s
	}

	for _, un := range pac.Unions {
		u := pac.newUnion(un)
		pac.unions[u.GoName()] = &u
	}
	return nil
}

func (pac *Package) write(g, c, h io.Writer) error {
	// C file
	fp(c, `#include "_cgo_export.h"`)
	fp(c, "")

	// Go file
	fp(g, "package ", pac.PacName)
	fp(g, "")
	fp(g, "/*")
	fp(g, "#include <", pac.From.File, ">")
	fp(g, `#include "`, path.Base(pac.hFile()), `"`)
	for _, d := range pac.From.CgoDirectives {
		fp(g, "#cgo ", d)
	}
	fp(g, "*/")
	fp(g, `import "C"`)
	fp(g, "")
	fp(g, "import (")
	fp(g, `"unsafe"`)
	for _, inc := range pac.Included {
		fp(g, `"`, inc.PacPath, `"`)
	}
	fp(g, ")")
	fp(g, "")

	cm := NewSSet()
	for _, fn := range pac.Functions {
		if len(fn.Ellipses) > 0 {
			continue
		}
		f := pac.newFunction(fn)
		if !pac.Exported(f) {
			continue
		}
		if info, ok := fn.HasCallback(); ok {
			// Go file
			callbackFunc := pac.newCallbackFunc(info)

			if !cm.Has(callbackFunc.goFuncName) {
				pac.callbacks = append(pac.callbacks, callbackFunc)
			}

			f1, f2 := pac.TransformOriginalFunc(fn, callbackFunc, info)
			if f.Receiver != nil {
				f1.Receiver.Object.AddMethod(f1)
				f2.Receiver.Object.AddMethod(f2)
			} else {
				pac.functions.Append(f1)
				pac.functions.Append(f2)
			}

			if !cm.Has(callbackFunc.goFuncName) {
				// H file
				fpn(h, "extern ")
				info.CType.WriteCDecl(h, callbackFunc.cFuncName)
				fp(h, ";")
				fp(h, "")

				// C file
				info.CType.WriteCallbackStub(c, callbackFunc.cFuncName, callbackFunc.goFuncName)
				fp(c, "")
			}

			// add into set
			cm.Add(callbackFunc.goFuncName)
		} else {
			if f.Receiver != nil {
				f.Receiver.Object.AddMethod(f)
			} else {
				pac.functions.Append(f)
			}
		}
	}

	for _, e := range pac.Enumerations {
		pac.define(g, pac.newEnum(e))
	}

	for _, v := range pac.Variables {
		pac.define(g, pac.newVariable(v))
	}

	{
		ns := make([]string, 0, len(pac.structs))
		for n := range pac.structs {
			ns = append(ns, n)
		}
		sort.Strings(ns)
		for _, n := range ns {
			s := pac.structs[n]
			if !pac.Exported(s) {
				continue
			}
			s.OptimizeNames()
			s.Define(g)
		}
	}

	{
		ns := make([]string, 0, len(pac.unions))
		for n := range pac.unions {
			ns = append(ns, n)
		}
		sort.Strings(ns)
		for _, n := range ns {
			u := pac.unions[n]
			if !pac.Exported(u) {
				continue
			}
			u.Define(g)
		}
	}

	for _, d := range pac.Typedefs {
		td := pac.newTypedef(d)
		if td.isValid() {
			pac.define(g, td)
		}
	}

	for _, f := range pac.functions {
		if !pac.Exported(f) {
			continue
		}
		f.Define(g)
	}

	for _, f := range pac.callbacks {
		f.Define(g)
	}

	p("Succesfully written to:")
	p(pac.goFile())
	p(pac.cFile())
	p(pac.hFile())
	pac.Statistics.Print()
	p()
	return nil
}

func (pac *Package) define(w io.Writer, t NamedType) {
	if !pac.Exported(t) {
		return
	}
	fp(w, "// ", t.CName())
	t.Define(w)
	fp(w, "")
	pac.DefCount++
}

func (pac *Package) Exported(t NamedType) bool {
	for _, n := range pac.From.Excluded {
		if n == t.CName() {
			return false
		}
	}
	return pac.fileIds.Has(t.File()) && pac.hasPrefix(t.CName())
}

// type name that may be declared in this or included packages.
func (pac *Package) globalName(o gcc.Named) string {
	if pac.fileIds.Has(o.File()) && pac.hasPrefix(o.CName()) {
		return pac.localName(o)
	}
	for _, inc := range pac.Included {
		if goName := inc.globalName(o); goName != "" && !contains(goName, ".") {
			return joins(inc.PacName, ".", goName)
		}
	}
	return ""
}

// upper name that is unique within the package
func (pac *Package) localName(o gcc.Named) string {
	n := pac.upperName(o)
	if sid, exists := pac.localNames[n]; !exists || o.Id() == sid {
		pac.localNames[n] = o.Id()
		return n
	}
	for {
		n += "_"
		if _, exists := pac.localNames[n]; !exists {
			break
		}
	}
	pac.localNames[n] = o.Id()
	return n
}

// upper camel name
func (pac *Package) upperName(o gcc.Named) string {
	return upperName(o.CName(), pac.From.NamePrefix)
}

// lower camel name
func (pac *Package) lowerName(o gcc.Named) string {
	s := snakeToLowerCamel(o.CName())
	switch s {
	case "type", "len":
		s += "_"
	}
	return s
}

func (pac *Package) hasPrefix(s string) bool {
	return hasPrefix(s, pac.From.NamePrefix)
}

func (pac *Package) isBool(cTypeName string) bool {
	return pac.boolSet.Has(cTypeName)
}

func (pac *Package) GenConst(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	ms, err := gcc.Xml{pac.From.FullPath()}.Macros()
	if err != nil {
		return err
	}
	consts := ms.Constants(pac.From.NamePrefix)

	fp(f, "package ", pac.PacName)
	fp(f, "")
	fp(f, "const (")
	for _, c := range consts {
		fp(f, snakeToCamel(trimPrefix(c.Name, pac.From.NamePrefix)), "=",
			snakeToCamel(remove(c.Body, pac.From.NamePrefix+"_")))
	}
	fp(f, ")")
	return nil
}

type Statistics struct {
	DefCount int
}

func (s Statistics) Print() {
	p(s.DefCount, "definitions wrapped.")
}
