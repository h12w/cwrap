// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"h12.io/cwrap"
	. "h12.io/cwrap"
)

const (
	HeaderDir = "/usr/local/include"
	// HeaderDir = "/usr/local/Cellar/netcdf/4.6.3_1/include"
	PacDir = "go-netcdf"
)

var (
	netcdf = &Package{
		PacName: "netcdf",
		PacPath: "go-netcdf",
		From: Header{
			Dir:         HeaderDir,
			File:        "netcdf.h",
			OtherCode:   "",
			NamePattern: `(?i:\Anc_(.*))`,
			Excluded: []string{
				"ncrecput", "nc_def_user_format", "nc_inq_user_format",
				"nc__open",
				"nc__enddef",
			},
			CgoDirectives: []string{"pkg-config: netcdf"},
			BoolTypes:     boolTypes,
		},
		TypeRule: typeRule,
		Included: []*Package{},
	}

	typeRule = map[string]string{}

	boolTypes = []string{
		// "cairo_bool_t",
	}
)

func Test(*testing.T) {
	OutputDir = "../../../"
	c(netcdf.Prepare())
	ncid := cwrap.NewSimpleTypeDef("", "FileID", 4)
	netcdf.TypeDeclMap["FileID"] = ncid
	var fs []*cwrap.Function
	for _, f := range netcdf.Functions {
		if len(f.CArgs) > 0 {
			first := f.CArgs[0]
			if first.CgoName() == "_ncid" {
				f.GoParams = f.GoParams[1:]
				m := &Method{Function: f, Receiver: ReceiverArg{Argument: cwrap.NewArgument("ncid", "_ncid", ncid), EqualType: ncid}}
				m.SetGoName(netcdf.UpperName(f.CName()))
				ncid.AddMethod(m)
				continue
			}
		}
		fs = append(fs, f)

	}
	netcdf.Functions = fs
	c(netcdf.Wrap())
}
