// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	. "h12.io/cwrap"
	"testing"
)

const (
	HeaderDir = "/usr/include/"
	PacDir    = "go-cairo/"
)

var (
	cairo = &Package{
		PacName: "cairo",
		PacPath: "go-cairo",
		From: Header{
			Dir:           HeaderDir,
			File:          "cairo/cairo.h",
			OtherCode:     "",
			NamePattern:   `(?i:\Acairo(.*))`,
			Excluded:      []string{},
			CgoDirectives: []string{"pkg-config: cairo"},
			BoolTypes:     boolTypes,
		},
		TypeRule: typeRule,
		Included: []*Package{},
	}

	typeRule = map[string]string{}

	boolTypes = []string{
		"cairo_bool_t",
	}
)

func Test(*testing.T) {
	//	OutputDir += "reg/"
	c(cairo.Wrap())
}
