// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	. "github.com/hailiang/cwrap"
	"testing"
)

/*
make mupdf with flags same as "go env GOGCCFLAGS", e.g.

    make XCFLAGS="-g -O2 -fPIC -m64 -pthread"

TODO: error handling of mupdf by weird setjmp/longjmp

*/

const (
	HeaderDir = "/usr/local/include/"
	PacDir    = "go-mupdf/"
	Ldflags   = "LDFLAGS: /usr/local/lib/libmupdf.a /usr/local/lib/libmupdf-js-none.a -lm -lfreetype -ljpeg -lopenjp2 -ljbig2dec -lssl -lcrypto"
)

var (
	pdf = &Package{
		PacName: "mupdf",
		PacPath: "go-mupdf",
		From: Header{
			Dir:         HeaderDir,
			File:        "mupdf/pdf.h",
			OtherCode:   "",
			NamePattern: `(?i:\Apdf(.*))`,
			Excluded: []string{ // functions of "undefined reference".
				"pdf_access_submit_event",
				"pdf_jsimp_array_item",
				"pdf_jsimp_array_len",
				"pdf_jsimp_from_number",
				"pdf_jsimp_from_string",
				"pdf_jsimp_new_obj",
				"pdf_jsimp_new_type",
				"pdf_jsimp_property",
				"pdf_jsimp_to_number",
				"pdf_jsimp_to_string",
				"pdf_jsimp_to_type",
				"pdf_new_jsimp",
				"pdf_open_compressed_stream",
				"pdf_crypt_buffer",
				"pdf_drop_jsimp",
				"pdf_init_ui_pointer_event",
				"pdf_jsimp_addmethod",
				"pdf_jsimp_addproperty",
				"pdf_jsimp_drop_obj",
				"pdf_jsimp_drop_type",
				"pdf_jsimp_execute",
				"pdf_jsimp_execute_count",
				"pdf_jsimp_set_global_type",
			},
			CgoDirectives: []string{Ldflags},
		},
		Included: []*Package{fz},
	}

	fz = &Package{
		PacName: "fz",
		PacPath: PacDir + "fz",
		From: Header{
			Dir:         HeaderDir,
			File:        "mupdf/fitz.h",
			NamePattern: `(?i:\Afz(.*))`,
			Excluded: []string{
				"fz_get_annot_type",
				"fz_open_file_w",
			},
			CgoDirectives: []string{Ldflags},
		},
		Included: []*Package{},
	}
)

func Test(*testing.T) {
	c(pdf.Wrap())
	c(fz.Wrap())
	//fz.DebugPrint()
}
