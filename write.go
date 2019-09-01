package cwrap

import (
	"io"
	"log"
	"path"
	"sort"
)

func (pac *Package) prepareFunctions() error {
	// populate functions/callbacks (and collect types)
	// write .h/.c directly
	var functions []*Function
	var callbacks []CallbackFunc
	callbackSet := NewSSet()
	for _, fn := range pac.XmlDoc.Functions {
		if len(fn.Ellipses) > 0 {
			continue
		}
		if !pac.exported(fn.CName(), fn.File()) {
			// log.Print("skip unexported function ", fn.CName(), " in ", fn.File())
			continue
		}
		f := pac.newFunction(fn)
		if info, ok := fn.HasCallback(); ok {
			// Go file
			callbackFunc := pac.newCallbackFunc(info)

			if !callbackSet.Has(callbackFunc.goName) {
				callbacks = append(callbacks, callbackFunc)
			}

			f1, f2 := pac.TransformOriginalFunc(fn, callbackFunc, info)
			functions = append(functions, f1)
			functions = append(functions, f2)

			if !callbackSet.Has(callbackFunc.goName) {

			}

			// add into set
			callbackSet.Add(callbackFunc.goName)
		} else {
			functions = append(functions, f)
		}
	}
	pac.Functions = functions
	pac.Callbacks = callbacks

	// populate variables (and collect types)
	variables := make([]*Variable, 0, len(pac.Variables))
	for _, v := range pac.XmlDoc.Variables {
		if pac.exported(v.CName(), v.File()) {
			variables = append(variables, pac.newVariable(v))
		}
	}
	pac.Variables = variables

	return nil
}

func (pac *Package) prepareTypesAndNames() {
	// populate fields (and collect types till no new types come out)
	for {
		cnt := len(pac.TypeDeclMap)
		pac.TypeDeclMap.Each(func(d TypeDecl) {
			switch s := d.(type) {
			case *Struct:
				if s.Fields == nil {
					s.Fields = pac.newStructFields(pac.FindStruct(s.Id()).Fields())
				}
			case *Union:
				if s.Fields == nil {
					s.Fields = pac.newUnionFields(pac.FindUnion(s.Id()).Fields(), s)
				}
			}
		})
		if cnt == len(pac.TypeDeclMap) {
			break
		}
	}

	excluded := []string{}

	// find linked list, remove the struct and keep the typedef, must go before
	// type names are assigned, so that the typedef can get proper names.
	pac.TypeDeclMap.Each(func(d TypeDecl) {
		if t, ok := d.(*Typedef); ok {
			if o, ok := pac.TypeDeclMap[t.rootId]; ok && o.CName() == t.CName() {
				excluded = append(excluded, t.rootId)
				t.id = t.rootId
			}
		}
	})

	// assign names to types, if empty, remove it.
	pac.TypeDeclMap.Each(func(d TypeDecl) {
		goName := pac.globalName(d)
		if goName != "" {
			d.SetGoName(goName)
		} else {
			excluded = append(excluded, d.Id())
		}
	})

	// assign names to functions (must go after types because type name of
	// receiver must be settled first.
	{
		var fs []*Function
		for _, f := range pac.Functions {
			if m, ok := f.ConvertToMethod(); ok {
				m.SetGoName(pac.UpperName(f.CName()))
			} else {
				f.SetGoName(pac.localName(f))
				fs = append(fs, f)
			}
		}
		pac.Functions = fs
	}

	// add all enumerations regardless of its appearance in functions
	for _, em := range pac.Enumerations {
		if pac.excluded(em.CName()) || !pac.included(em.File()) {
			continue
		}
		e := pac.declareEqualType(em).(*Enum)
		goName := pac.globalName(e)
		if goName != "" {
			e.goName = goName
		}
		for i, v := range e.Values {
			e.Values[i].goName = pac.globalName(v)
		}
	}
	// then remove the enumeration if it is typedefed.
	pac.TypeDeclMap.Each(func(d TypeDecl) {
		if t, ok := d.(*Typedef); ok {
			if e, ok := t.Literal.(*Enum); ok {
				excluded = append(excluded, e.Id())
			}
		}
	})

	// optimize 2nd level names like fields and methods, must go after all
	// global level names are settled.
	pac.TypeDeclMap.Each(func(d TypeDecl) {
		if o, ok := d.(NameOptimizer); ok {
			o.OptimizeNames()
		}
	})

	// assign name to variables
	for _, v := range pac.Variables {
		v.SetGoName(pac.localName(v))
	}

	// remove excluded types
	for _, id := range excluded {
		pac.TypeDeclMap.Delete(id)
	}
}

// Huge function to write all the stuff
func (pac *Package) write() error {
	if len(pac.Callbacks) > 0 {
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
		if err := pac.writeCFile(c, h); err != nil {
			return err
		}
		log.Print("written to ", pac.cFile())
		log.Print("written to ", pac.hFile())
	}

	g, err := pac.createFile(pac.goFile())
	if err != nil {
		return err
	}
	defer g.Close()
	if err := pac.writeGoFile(g); err != nil {
		return err
	}
	if err := gofmt(pac.goFile()); err != nil {
		return err
	}

	log.Print("written to ", pac.goFile())
	pac.Statistics.Print()
	p()

	return nil
}

func (pac *Package) writeCFile(c, h io.Writer) error {
	// C file starts
	fp(c, `#include "_cgo_export.h"`)
	fp(c, "")

	for _, callbackFunc := range pac.Callbacks {
		// H file
		fpn(h, "extern ")
		callbackFunc.CType.WriteCDecl(h, callbackFunc.cFuncName)
		fp(h, ";")
		fp(h, "")

		// C file
		callbackFunc.CType.WriteCallbackStub(c, callbackFunc.cFuncName, callbackFunc.goName)
		fp(c, "")
	}
	return nil
}

func (pac *Package) writeGoFile(g io.Writer) error {
	// Go file starts
	fp(g, "package ", pac.PacName)
	fp(g, "")
	fp(g, "/*")
	fp(g, "#include <", pac.From.File, ">")
	if len(pac.Callbacks) > 0 {
		fp(g, `#include "`, path.Base(pac.hFile()), `"`)
	}
	fp(g, "#include <stdlib.h>")
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

	for _, v := range pac.Variables {
		pac.writeDecl(g, "var", v)
	}

	ds := pac.TypeDeclMap.ToSlice()
	for _, d := range ds {
		pac.writeDecl(g, "type", d)
		fp(g, "")
	}

	for _, f := range pac.Functions {
		pac.writeDecl(g, "func", f)
	}

	for _, f := range pac.Callbacks {
		f.Declare(g)
	}

	return nil
}

func (pac *Package) writeDecl(w io.Writer, keyword string, d Decl) {
	if pac.excluded(d.CName()) || contains(d.GoName(), ".") {
		return
	}
	if d.GoName() != "" &&
		d.Id() != "" { // is not simple typedef
		fp(w, "// ", d.CName())
		fpn(w, keyword, " ", d.GoName(), " ")
		d.WriteSpec(w)
	}
	if t, ok := d.(TypeDecl); ok {
		t.WriteMethods(w)
	}
	fp(w, "")
	pac.DefCount++
}

// type name that may be declared in this or included packages.
func (pac *Package) globalName(o CNamer) string {
	if pac.fileIds.Has(o.File()) && pac.matched(o.CName()) {
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
func (pac *Package) localName(o CNamer) string {
	n := pac.UpperName(o.CName())
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
func (pac *Package) UpperName(cName string) string {
	return upperName(cName, pac.pat)
}

func (pac *Package) isBool(cTypeName string) bool {
	return pac.boolSet.Has(cTypeName)
}

func (pac *Package) declare(d TypeDecl) {
	pac.TypeDeclMap[d.Id()] = d
}

func (pac *Package) excluded(cName string) bool {
	for _, n := range pac.From.Excluded {
		if n == cName {
			return true
		}
	}
	return false
}

func (pac *Package) included(file string) bool {
	return pac.fileIds.Has(file)
}

func (pac *Package) matched(cName string) bool {
	return pac.pat.MatchString(cName)
}

func (pac *Package) exported(cName, file string) bool {
	return !pac.excluded(cName) &&
		pac.included(file) &&
		pac.matched(cName)
}

type TypeDecls []TypeDecl

func (s TypeDecls) Len() int {
	return len(s)
}

func (s TypeDecls) Less(i, j int) bool {
	ni := s[i].GoName()
	if ni == "" {
		ni = s[i].CName()
	}
	nj := s[j].GoName()
	if nj == "" {
		nj = s[j].CName()
	}
	return ni < nj
}

func (s TypeDecls) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type TypeDeclMap map[string]TypeDecl

func (m TypeDeclMap) Delete(id string) {
	delete(m, id)
}

func (m TypeDeclMap) ToSlice() TypeDecls {
	ds := make(TypeDecls, 0, len(m))
	for _, d := range m {
		ds = append(ds, d)
	}
	sort.Sort(ds)
	return ds
}

func (m TypeDeclMap) Each(visit func(d TypeDecl)) {
	for _, d := range m {
		eachDecl(d, visit)
	}
}

func eachDecl(d TypeDecl, visit func(TypeDecl)) {
	visit(d)
	if t, ok := d.(*Typedef); ok {
		if dd, ok := t.Literal.(TypeDecl); ok {
			eachDecl(dd, visit)
		}
	}
}
