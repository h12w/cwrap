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
	"path/filepath"
	"regexp"
	"runtime"
)

var (
	GOPATHs   = filepath.SplitList(os.Getenv("GOPATH"))
	OutputDir = GOPATHs[0] + "/src/"
)

type Header struct {
	Dir         string
	File        string
	NamePattern string
	OtherCode   string
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
	TypeRule map[string]string
	ArgRule  map[string]string

	// Internal
	pat         *regexp.Regexp
	localNames  map[string]string
	fileIds     SSet
	boolSet     SSet
	typeDeclMap TypeDeclMap
	Statistics
	*gcc.XmlDoc
}

func (pac *Package) Load() (err error) {
	if pac.From.NamePattern == "" {
		pac.From.NamePattern = ".*"
	}
	pac.pat = regexp.MustCompile(pac.From.NamePattern)
	pac.localNames = make(map[string]string)
	pac.initBoolSet()
	pac.typeDeclMap = make(TypeDeclMap)
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
	// reset localNames
	pac.localNames = make(map[string]string)
	return nil
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
	consts := ms.Constants(pac.From.NamePattern)
	nm := make(map[string]string)
	for _, c := range consts {
		nm[c.Name] = upperName(c.Name, pac.pat)
	}

	fp(f, "package ", pac.PacName)
	fp(f, "")
	fp(f, "const (")
	for _, c := range consts {
		body := c.Body
		for k, v := range nm {
			body = replace(body, k, v)
		}
		fp(f, upperName(c.Name, pac.pat), "=", body)
	}
	fp(f, ")")
	return nil
}

type Statistics struct {
	DefCount int
}

func (s Statistics) Print() {
	p(s.DefCount, "declarations wrapped.")
}
