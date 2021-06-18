// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	. "h12.io/cwrap"
)

const (
	HeaderDir = "/usr/local/include/"
)

var (
	paho = &Package{
		PacName: "paho",
		PacPath: "paho",
		From: Header{
			Dir:           HeaderDir,
			File:          "MQTTAsync.h",
			NamePattern:   `\AMQTT(?:Async)?(.*)`,
			Excluded:      []string{"MQTTProperties_len", "MQTTProperties_read", "MQTTProperties_write"},
			CgoDirectives: []string{"LDFLAGS: -lpaho-mqtt3a"},
		},
		Included: []*Package{},
	}
)

func Test(*testing.T) {
	c(paho.Wrap())
}
