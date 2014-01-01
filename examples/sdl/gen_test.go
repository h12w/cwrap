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
			Dir:        HeaderDir,
			File:       "SDL2/SDL.h",
			OtherCode:  "#define _SDL_main_h",
			NamePrefix: "SDL",
			Excluded: []string{
				"SDL_LogMessageV",
				"SDL_vsnprintf",
				"SDL_ThreadID",
				"SDL_GetThreadID",
			},
			CgoDirectives: []string{"pkg-config: sdl2"},
		},
		Included: []*Package{},
	}

	ttf = &Package{
		PacName: "ttf",
		PacPath: PacDir + "ttf",
		From: Header{
			Dir:           HeaderDir,
			File:          "SDL2/SDL_ttf.h",
			NamePrefix:    "TTF",
			CgoDirectives: []string{"pkg-config: SDL2_ttf"},
		},
		Included: []*Package{sdl},
	}
)

func Test(*testing.T) {
	c(ttf.Wrap())
	c(sdl.Wrap())
}
