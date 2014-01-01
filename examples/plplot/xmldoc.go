// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var (
	directionRe = regexp.MustCompile(`(?s).*(input|output).*`)
	spacesRe    = regexp.MustCompile(`(?s)\s+`)
	refRe       = regexp.MustCompile(`&\w+;`)
)

type Chapter struct {
	Sections []Section `xml:"sect1"`
}

type Section struct {
	Title          Title          `xml:"title"`
	Function       string         `xml:"para>funcsynopsis>funcprototype>funcdef>function"`
	VarListEntries []VarListEntry `xml:"variablelist>varlistentry"`
}

type Title struct {
	CharData string `xml:",chardata"`
}

type VarListEntry struct {
	Term Term             `xml:"term"`
	Para VarListEntryPara `xml:"listitem>para"`
}

type VarListEntryPara struct {
	CharData string `xml:",chardata"`
}

type Term struct {
	Argument string `xml:"parameter"`
	Literal  string `xml:"literal"`
	CharData string `xml:",chardata"`
}

type Function struct {
	Name string
	Args []Argument
	Doc  string
}

type Direction int

const (
	In Direction = iota
	Out
)

type Argument struct {
	Name string
	Di   Direction
	Doc  string
}

func parseApiXml(file, prefix string) []Function {
	f, err := os.Open(file)
	c(err)
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	c(err)
	buf = escape(buf, prefix)
	var chap Chapter
	c(xml.Unmarshal(buf, &chap))
	functions := make([]Function, len(chap.Sections))
	for i, section := range chap.Sections {
		functions[i] = Function{
			Name: section.Function,
			Args: parseArgs(section.VarListEntries),
			Doc:  cleanText(section.Title.CharData),
		}
	}
	return functions
}

func parseArgs(entries []VarListEntry) (params []Argument) {
	for _, entry := range entries {
		direction := In
		m := directionRe.FindStringSubmatch(entry.Term.CharData)
		if len(m) > 0 && m[1] == "output" {
			direction = Out
		}
		paramNames := strings.Split(entry.Term.Argument, ",")
		for _, paramName := range paramNames {
			params = append(params, Argument{
				Name: strings.TrimSpace(paramName),
				Di:   direction,
				Doc:  cleanText(entry.Para.CharData),
			})
		}
	}
	return
}

func cleanText(s string) string {
	s = spacesRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, ": ")
	return s
}

// Change docbook function reference to Go names but keeps XML escape chars
// intact (e.g. from &plwind; to function Wind).
func escape(buf []byte, prefix string) []byte {
	return refRe.ReplaceAllFunc(buf, func(ref []byte) []byte {
		refName := string(ref[1 : len(ref)-1])
		switch refName {
		case "lt", "gt", "amp":
			return ref
		}
		return []byte("function " + toGoFuncName(refName, prefix))
	})
}

func toGoFuncName(name, prefix string) string {
	name = strings.TrimPrefix(name, prefix)
	name = strings.TrimPrefix(name, "_")
	return strings.Title(name)
}
