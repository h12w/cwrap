// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/hailiang/cwrap"
	"testing"
)

const (
	HeaderDir     = "/usr/local/include/"
	PacDir        = "go-plplot/"
)

var (
	plplot = &cwrap.Package{
		PacName: "c",
		PacPath: PacDir + "c",
		From: cwrap.Header{
			Dir:        "/usr/local/include/",
			File:       "plplot/plplot.h",
			NamePrefix: "c_pl",
			BoolTypes:  []string{"PLBOOL"},
			Excluded: []string{
				"c_plwid",
				"c_pltimefmt",
				"c_plssub",
				"c_plwidth",
				"c_pllsty",
//				"c_plcont",
//				"c_pllegend",
			},
			CgoDirectives: []string{"pkg-config: plplotd"},
		},
	}
)

func Test(*testing.T) {
	c(plplot.Wrap())
}
