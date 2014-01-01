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
	"syscall"
)

var (
	GOPATH, _ = syscall.Getenv("GOPATH")
	OutputDir = GOPATH + "/src/"
)

type Header struct {
	Dir           string
	File          string
	OtherCode     string
	NamePrefix    string
	// Not define it in the package, but may still searchable as included types
	// because it may be manually defined.
	Excluded      []string
	CgoDirectives []string
	BoolTypes     []string
}

func (h Header) FullPath() string {
	return h.Dir + h.File
}

func (h Header) Write(w io.Writer) {
	fp(w, h.OtherCode)
	fp(w, "#include <", h.File, ">")
}

type Package struct {
	PacName    string
	PacPath    string
	From       Header
	Included   []*Package
	GoFile     string
	CFile      string
	HFile      string
	localNames map[string]string
	fileIds    SSet
	boolSet    SSet
	Statistics
	*gcc.XmlDoc
}

func (pac *Package) Load() (err error) {
	pac.localNames = make(map[string]string)
	pac.initBoolSet()
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
	if err := pac.write(g, c, h); err != nil {
		return err
	}
	return gofmt(pac.goFile())
}

func (pac *Package) write(g, c, h io.Writer) error {
	if pac.XmlDoc == nil {
		if err := pac.Load(); err != nil {
			return err
		}
	}
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

	for _, fn := range pac.Functions {
		if len(fn.Ellipses) > 0 {
			continue
		}
		// register name
		pac.globalName(fn)
	}

	for _, e := range pac.Enumerations {
		pac.define(g, pac.newEnum(e))
	}

	for _, v := range pac.Variables {
		pac.define(g, pac.newVariable(v))
	}

	for _, s := range pac.Structs {
		pac.define(g, pac.newStruct(s))
	}

	for _, s := range pac.Unions {
		pac.define(g, pac.newUnion(s))
	}

	for _, d := range pac.Typedefs {
		td := pac.newTypedef(d)
		if td.isValid() {
			pac.define(g, td)
		}
	}

	cm := NewSSet()
	for _, fn := range pac.Functions {
		if len(fn.Ellipses) > 0 {
			continue
		}
		if !pac.exported(pac.newFunction(fn)) {
			continue
		}
		f := pac.newFunction(fn)
		if info, ok := fn.HasCallback(); ok {
			// Go file
			callbackFunc := pac.newCallbackFunc(info)

			if !cm.Has(callbackFunc.goFuncName) {
				callbackFunc.Define(g)
				fp(g, "")
			}

			transFunc := callbackFunc.TransformOriginalFunc(f, info)
			transFunc.Define(g)
			fp(g, "")
			f = pac.newFunction(fn)
			f.goName += "_"
			pac.define(g, f)

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
			pac.define(g, f)
		}
	}

	p("Succesfully written to:")
	p(pac.goFile())
	p(pac.cFile())
	p(pac.hFile())
	pac.Statistics.Print()
	p()
	return nil
}

func (pac *Package) exported(t NamedType) bool {
	return pac.fileIds.Has(t.File()) && pac.hasPrefix(t.CName())
}

func (pac *Package) define(w io.Writer, t NamedType) {
	if pac.exported(t) {
		if pac.Excluded(t.CName()) {
			return
		}
		fp(w, "// ", t.CName())
		t.Define(w)
		fp(w, "")
		pac.DefCount++
	}
}

func (pac *Package) Excluded(name string) bool {
	for _, n := range pac.From.Excluded {
		if n == name {
			return true
		}
	}
	return false
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
