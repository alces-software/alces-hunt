// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package output

import (
	"strings"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

func TestListPlainParsed(t *testing.T) {
	n := node.New("a1b2c3d4e5f6", "node01", "10.0.0.41", "---\n", []string{"gpu", "compute"}, map[string]string{"label": "cnode041"})
	n.Label = "cnode041"
	got := ListPlain(n)
	fields := strings.Split(got, "\t")
	if len(fields) != 6 {
		t.Fatalf("want 6 fields, got %d: %q", len(fields), got)
	}
	if fields[0] != "a1b2c3d4e5f6" || fields[1] != "node01" || fields[2] != "10.0.0.41" {
		t.Fatalf("identity fields: %v", fields[:3])
	}
	if fields[3] != "compute|gpu" {
		t.Fatalf("groups %q", fields[3])
	}
	if fields[4] != "cnode041" {
		t.Fatalf("label %q", fields[4])
	}
	if fields[5] != `{"label":"cnode041"}` {
		t.Fatalf("presets %q", fields[5])
	}
	if strings.Contains(fields[5], " ") {
		t.Fatal("presets JSON must be compact")
	}
}

func TestListPlainBufferEmpty(t *testing.T) {
	n := node.New("a1b2c3d4e5f6", "node01", "10.0.0.41", "", nil, nil)
	got := ListPlain(n)
	fields := strings.Split(got, "\t")
	if fields[3] != "|" {
		t.Fatalf("empty groups want | got %q", fields[3])
	}
	if fields[4] != "" {
		t.Fatalf("empty label want empty got %q", fields[4])
	}
	if fields[5] != "{}" {
		t.Fatalf("empty presets want {{}} got %q", fields[5])
	}
}

func TestShowPlainContentNewlines(t *testing.T) {
	content := "---\nsysuuid: abc123\nbootif: 01-00-11-22-33-44-55\n"
	n := node.New("id1", "node01", "10.0.0.41", content, []string{"compute", "gpu"}, nil)
	got := ShowPlain(n)
	fields := strings.SplitN(got, "\t", 5)
	if len(fields) != 5 {
		t.Fatalf("want 5 fields, got %d", len(fields))
	}
	if fields[3] != "compute|gpu" {
		t.Fatalf("groups %q", fields[3])
	}
	if fields[4] != content {
		t.Fatalf("content not preserved: %q", fields[4])
	}
}

func TestListPlainNoHeader(t *testing.T) {
	n := node.New("id1", "h", "1.2.3.4", "", nil, nil)
	got := ListPlainAll([]*node.Node{n})
	if strings.Contains(strings.ToLower(got), "hostname") {
		t.Fatal("plain output must not include headers")
	}
}
