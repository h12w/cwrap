// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"

	gcc "h12.io/go-gccxml"
)

const tempDir = "./temp/"

var (
	GOPATHs   = filepath.SplitList(os.Getenv("GOPATH"))
	OutputDir = func() string {
		if len(GOPATHs) > 0 {
			return GOPATHs[0] + "/src/"
		}
		return "."
	}()
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
	GccXmlArgs    []string
}

func (h Header) FullPath() string {
	file := path.Join(h.Dir, h.File)
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

	// intermediate
	Functions   []*Function
	Callbacks   []CallbackFunc
	TypeDeclMap TypeDeclMap
	Variables   []*Variable

	// Internal
	pat        *regexp.Regexp
	localNames map[string]string
	fileIds    SSet
	boolSet    SSet
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
	pac.TypeDeclMap = make(TypeDeclMap)
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
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	f, err := ioutil.TempFile(tempDir, "_cwrap-*.h")
	if err != nil {
		return Wrap(err)
	}
	for _, inc := range pac.Included {
		inc.From.Write(f)
	}
	pac.From.Write(f)
	f.Close()
	xmlDocCfg := gcc.Xml{File: f.Name(), Args: pac.From.GccXmlArgs, CastXml: true}
	xmlDoc, err := xmlDocCfg.Doc()
	if err != nil {
		return err
	}
	pac.XmlDoc = xmlDoc
	xmlFile, err := os.Create(path.Join(tempDir, path.Base(f.Name())+".xml"))
	if err != nil {
		return err
	}
	defer xmlFile.Close()
	if err := xmlDocCfg.Save(xmlFile); err != nil {
		return err
	}
	return nil
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
	return path.Join(OutputDir, pac.PacPath, "/auto_"+runtime.GOARCH)
}

func (pac *Package) createFile(file string) (io.WriteCloser, error) {
	if err := os.MkdirAll(path.Dir(file), 0755); err != nil {
		return nil, Wrap(err)
	}
	f, err := os.Create(file)
	if err != nil {
		return nil, Wrap(err)
	}
	return f, nil
}

func (pac *Package) Wrap() error {
	if err := pac.Prepare(); err != nil {
		return err
	}
	if err := pac.write(); err != nil {
		return err
	}
	return nil
}

func (pac *Package) Prepare() error {
	if pac.XmlDoc == nil {
		if err := pac.Load(); err != nil {
			return err
		}
		if err := pac.prepareFunctions(); err != nil {
			return err
		}
		pac.prepareTypesAndNames()
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
	ms, err := gcc.Xml{File: pac.From.FullPath(), Args: pac.From.GccXmlArgs, CastXml: true}.Macros()
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
	log.Print(s.DefCount, " declarations wrapped.")
}
