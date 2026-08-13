// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package labels

import (
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

func TestHostnameLabel(t *testing.T) {
	cases := []struct {
		host, mode, want string
	}{
		{"node01.cluster.local", "long", "node01.cluster.local"},
		{"node01.cluster.local", "short", "node01"},
		{"node01.cluster.local", "blank", ""},
		{"node01", "short", "node01"},
	}
	for _, c := range cases {
		if got := HostnameLabel(c.host, c.mode); got != c.want {
			t.Errorf("HostnameLabel(%q,%q)=%q want %q", c.host, c.mode, got, c.want)
		}
	}
}

func TestAutoLabelPrefixPadding(t *testing.T) {
	n := node.New("id1", "host.example", "10.0.0.1", "", nil, nil)
	opt := Options{Prefix: "cnode", Start: "001", DefaultLabel: "long", DefaultStart: "01"}
	got := AutoLabel(n, nil, opt)
	if got != "cnode001" {
		t.Fatalf("got %q want cnode001", got)
	}
	got = AutoLabel(n, []string{"cnode001", "cnode002"}, opt)
	if got != "cnode003" {
		t.Fatalf("got %q want cnode003", got)
	}
}

func TestAutoLabelPrefixStartsOverride(t *testing.T) {
	n := node.New("id1", "host", "10.0.0.1", "", nil, map[string]string{"prefix": "cnode"})
	opt := Options{
		Prefix:       "ignored",
		Start:        "01",
		DefaultStart: "01",
		PrefixStarts: map[string]string{"cnode": "010"},
	}
	got := AutoLabel(n, nil, opt)
	if got != "cnode010" {
		t.Fatalf("got %q want cnode010", got)
	}
}

func TestAutoLabelPresetPrefixWinsOverCLI(t *testing.T) {
	n := node.New("id1", "host", "10.0.0.1", "", nil, map[string]string{"prefix": "gpu"})
	opt := Options{Prefix: "cnode", Start: "01", DefaultStart: "01"}
	got := AutoLabel(n, nil, opt)
	if got != "gpu01" {
		t.Fatalf("got %q want gpu01", got)
	}
}

func TestAutoLabelNoPrefixUsesHostname(t *testing.T) {
	n := node.New("id1", "login.example.com", "10.0.0.1", "", nil, nil)
	if got := AutoLabel(n, nil, Options{DefaultLabel: "short"}); got != "login" {
		t.Fatalf("got %q", got)
	}
}

func TestPresetLabelWins(t *testing.T) {
	n := node.New("id1", "host", "10.0.0.1", "", nil, map[string]string{"label": "foobar01", "prefix": "cnode"})
	if n.PresetLabel() != "foobar01" {
		t.Fatalf("preset label %q", n.PresetLabel())
	}
}

func TestFirstDuplicate(t *testing.T) {
	if FirstDuplicate([]string{"a", "b", "a"}) != "a" {
		t.Fatal("expected a")
	}
	if FirstDuplicate([]string{"a", "b"}) != "" {
		t.Fatal("expected empty")
	}
}
