// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	. "h12.io/cwrap"
	"testing"
)

const (
	HeaderDir    = "/usr/include/"
	PacDir       = "go-gsl/"
	CgoDirective = "LDFLAGS: -lgsl -lgslcblas"
)

var (
	rng = &Package{
		PacName: "rng",
		PacPath: PacDir + "rng",
		From: Header{
			Dir:           HeaderDir,
			File:          "gsl/gsl_rng.h",
			OtherCode:     "",
			NamePattern:   `\Agsl_rng(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{CgoDirective},
		},
		Included: []*Package{},
	}

	ran = &Package{
		PacName: "ran",
		PacPath: PacDir + "ran",
		From: Header{
			Dir:           HeaderDir,
			File:          "gsl/gsl_randist.h",
			OtherCode:     "",
			NamePattern:   `\Agsl_ran(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{CgoDirective},
		},
		Included: []*Package{rng},
	}

	stats = &Package{
		PacName: "stats",
		PacPath: PacDir + "stats",
		From: Header{
			Dir:           HeaderDir,
			File:          "gsl/gsl_statistics_double.h",
			OtherCode:     "",
			NamePattern:   `\Agsl_stats(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{CgoDirective},
		},
		Included: []*Package{},
	}

	fit = &Package{
		PacName: "fit",
		PacPath: PacDir + "fit",
		From: Header{
			Dir:           HeaderDir,
			File:          "gsl/gsl_fit.h",
			OtherCode:     "",
			NamePattern:   `\Agsl_fit(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{CgoDirective},
		},
		Included: []*Package{},
	}
)

func Test(*testing.T) {
	//OutputDir += "reg/"
	c(ran.Wrap())
	c(rng.Wrap())
	c(stats.Wrap())
	c(fit.Wrap())
}
