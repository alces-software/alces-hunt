// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := node.New("abc", "host01", "10.0.0.1", "---\nhello:\tworld\n", []string{"gpu", "compute"}, map[string]string{"label": "n01"})
	n.Label = "n01"
	l.Add(n)
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "abc.yaml")); err != nil {
		t.Fatal(err)
	}
	l2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := l2.FindByID("abc")
	if got == nil || got.Label != "n01" || got.Content != n.Content {
		t.Fatalf("round trip failed: %+v", got)
	}
	if got.PrettyGroups() != "compute, gpu" {
		t.Fatalf("groups %q", got.PrettyGroups())
	}
}

func TestReplaceByID(t *testing.T) {
	dir := t.TempDir()
	l, _ := Load(dir)
	l.Add(node.New("id1", "old", "1.1.1.1", "old", nil, nil))
	_ = l.Save()
	n := node.New("id1", "new", "2.2.2.2", "new", nil, nil)
	if err := l.ReplaceByID(n); err != nil {
		t.Fatal(err)
	}
	_ = l.Save()
	l2, _ := Load(dir)
	if len(l2.Nodes()) != 1 || l2.Nodes()[0].Hostname != "new" {
		t.Fatalf("replace failed: %+v", l2.Nodes())
	}
}

func TestEmpty(t *testing.T) {
	dir := t.TempDir()
	l, _ := Load(dir)
	l.Add(node.New("id1", "h", "1.1.1.1", "", nil, nil))
	_ = l.Save()
	if err := l.Empty(); err != nil {
		t.Fatal(err)
	}
	l2, _ := Load(dir)
	if len(l2.Nodes()) != 0 {
		t.Fatalf("expected empty, got %d", len(l2.Nodes()))
	}
}

func TestSortedByID(t *testing.T) {
	dir := t.TempDir()
	l, _ := Load(dir)
	l.Add(node.New("b", "h", "1", "", nil, nil))
	l.Add(node.New("a", "h", "1", "", nil, nil))
	nodes := l.Nodes()
	if nodes[0].ID != "a" || nodes[1].ID != "b" {
		t.Fatalf("not sorted: %s %s", nodes[0].ID, nodes[1].ID)
	}
}
