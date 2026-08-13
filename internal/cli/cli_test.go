// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package cli

import (
	"flag"
	"testing"
)

func TestParseArgsFlagsAfterPositional(t *testing.T) {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	var plain, buffer bool
	fs.BoolVar(&plain, "plain", false, "")
	fs.BoolVar(&buffer, "buffer", false, "")
	pos, err := parseArgs(fs, []string{"cnode001", "--plain"})
	if err != nil {
		t.Fatal(err)
	}
	if !plain || len(pos) != 1 || pos[0] != "cnode001" {
		t.Fatalf("plain=%v pos=%v", plain, pos)
	}
}

func TestParseArgsValueFlagAfterPositional(t *testing.T) {
	fs := flag.NewFlagSet("modify-groups", flag.ContinueOnError)
	var add string
	fs.StringVar(&add, "add", "", "")
	pos, err := parseArgs(fs, []string{"n01", "--add", "compute,gpu"})
	if err != nil {
		t.Fatal(err)
	}
	if add != "compute,gpu" || pos[0] != "n01" {
		t.Fatalf("add=%q pos=%v", add, pos)
	}
}
