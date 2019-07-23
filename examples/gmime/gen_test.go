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
	PacDir    = "go-gmime/"
)

var (
	gmime = &Package{
		PacName: "gmime",
		PacPath: "go-gmime",
		From: Header{
			Dir:           HeaderDir,
			File:          "gmime-2.6/gmime/gmime.h",
			NamePattern:   `\Ag_?mime(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{"pkg-config: gmime-2.6"},
			BoolTypes:     boolTypes,
			GccXmlArgs: []string{
				"-D_LARGEFILE64_SOURCE",
				"-I/usr/include/glib-2.0",
				"-I/usr/lib/x86_64-linux-gnu/glib-2.0/include",
				"-I/usr/include/gmime-2.6",
			},
		},
		TypeRule: typeRule,
		Included: []*Package{},
	}

	typeRule = map[string]string{}

	boolTypes = []string{
		"gboolean",
	}
)

func Test(*testing.T) {
	//	OutputDir += "reg/"
	c(gmime.Wrap())
	//gmime.GenConst("/dev/stdout")
}
