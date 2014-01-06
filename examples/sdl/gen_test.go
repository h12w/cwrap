// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	. "github.com/hailiang/cwrap"
	"testing"
)

const (
	HeaderDir = "/usr/local/include/"
	PacDir    = "go-sdl/"
)

var (
	sdl = &Package{
		PacName: "sdl",
		PacPath: "go-sdl",
		From: Header{
			Dir:           HeaderDir,
			File:          "SDL2/SDL.h",
			OtherCode:     "#define _SDL_main_h",
			NamePattern:   `\ASDL(.*)`,
			Excluded:      []string{},
			CgoDirectives: []string{"pkg-config: sdl2"},
			BoolTypes:     boolTypes,
		},
		TypeRule: typeRule,
		Included: []*Package{},
	}

	ttf = &Package{
		PacName: "ttf",
		PacPath: PacDir + "ttf",
		From: Header{
			Dir:           HeaderDir,
			File:          "SDL2/SDL_ttf.h",
			NamePattern:   `\ATTF(.*)`,
			CgoDirectives: []string{"pkg-config: SDL2_ttf"},
			BoolTypes:     boolTypes,
		},
		TypeRule: typeRule,
		Included: []*Package{sdl},
	}

	typeRule = map[string]string{
		"Uint8":  "byte",
		"Uint16": "uint16",
		"Uint32": "uint32",
		"Uint64": "uint64",
		"Sint8":  "int8",
		"Sint16": "int16",
		"Sint32": "int32",
		"Sint64": "int64",
	}

	boolTypes = []string{
		"SDL_bool",
	}
)

func Test(*testing.T) {
//	OutputDir += "reg/"
	c(ttf.Wrap())
	c(sdl.Wrap())
	//ttf.GenConst("/dev/stdout")
	//sdl.GenConst("/dev/stdout")
}
