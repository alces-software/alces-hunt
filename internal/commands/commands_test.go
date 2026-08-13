// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package commands

import (
	"strings"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/selectnode"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

func cfgRoot(t *testing.T) *config.Config {
	t.Helper()
	root := t.TempDir()
	t.Setenv("ALCES_HUNT_ROOT", root)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func addParsed(t *testing.T, cfg *config.Config, nodes ...*node.Node) {
	t.Helper()
	list, err := store.Load(cfg.ParsedDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		list.Add(n)
	}
	if err := list.Save(); err != nil {
		t.Fatal(err)
	}
}

func addBuffer(t *testing.T, cfg *config.Config, nodes ...*node.Node) {
	t.Helper()
	list, err := store.Load(cfg.BufferDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		list.Add(n)
	}
	if err := list.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveByLabelAndBufferID(t *testing.T) {
	cfg := cfgRoot(t)
	n := node.New("abc", "host01", "10.0.0.1", "", []string{"compute"}, nil)
	n.Label = "cnode001"
	addParsed(t, cfg, n)
	if err := RemoveNode(cfg, "cnode001", SelectOptions{}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	if len(list.Nodes()) != 0 {
		t.Fatal("parsed node should be gone")
	}

	b := node.New("def", "host02", "10.0.0.2", "", nil, nil)
	addBuffer(t, cfg, b)
	if err := RemoveNode(cfg, "cnode001", SelectOptions{Buffer: true}); err == nil {
		t.Fatal("label must not match buffer")
	}
	if err := RemoveNode(cfg, "def", SelectOptions{Buffer: true}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveMatchHostname(t *testing.T) {
	cfg := cfgRoot(t)
	n := node.New("abc", "login01", "10.0.0.1", "", nil, nil)
	n.Label = "cnode001"
	addParsed(t, cfg, n)
	if err := RemoveNode(cfg, "login01", SelectOptions{MatchHostname: true}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	if len(list.Nodes()) != 0 {
		t.Fatal("should have matched hostname")
	}
}

func TestRemoveRegex(t *testing.T) {
	cfg := cfgRoot(t)
	a := node.New("a", "h", "1.1.1.1", "", nil, nil)
	a.Label = "cnode001"
	b := node.New("b", "h", "1.1.1.2", "", nil, nil)
	b.Label = "login01"
	addParsed(t, cfg, a, b)
	if err := RemoveNode(cfg, "^cnode", SelectOptions{Regex: true}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	if len(list.Nodes()) != 1 || list.Nodes()[0].Label != "login01" {
		t.Fatalf("regex remove failed: %+v", list.Nodes())
	}
}

func TestRemoveGenders(t *testing.T) {
	cfg := cfgRoot(t)
	for _, name := range []string{"node1", "node2", "node3"} {
		n := node.New(name, name, "1.1.1.1", "", nil, nil)
		n.Label = name
		addParsed(t, cfg, n)
	}
	if err := RemoveNode(cfg, "node[1-2]", SelectOptions{}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	if len(list.Nodes()) != 1 || list.Nodes()[0].Label != "node3" {
		t.Fatalf("got %+v", list.Nodes())
	}
}

func TestModifyAndRenameGroups(t *testing.T) {
	cfg := cfgRoot(t)
	n := node.New("abc", "h", "1.1.1.1", "", []string{"old"}, nil)
	n.Label = "n01"
	addParsed(t, cfg, n)
	if err := ModifyGroups(cfg, "n01", ModifyGroupsOptions{Add: "compute,gpu", Remove: "old"}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	got := list.Nodes()[0].SortedGroups()
	if strings.Join(got, ",") != "compute,gpu" {
		t.Fatalf("groups %v", got)
	}
	if err := RenameGroup(cfg, "compute", "batch", false); err != nil {
		t.Fatal(err)
	}
	list, _ = store.Load(cfg.ParsedDir())
	if !list.Nodes()[0].HasGroup("batch") || list.Nodes()[0].HasGroup("compute") {
		t.Fatalf("rename failed %v", list.Nodes()[0].Groups)
	}
	if err := RenameGroup(cfg, "missing", "x", false); err == nil {
		t.Fatal("expected missing group error")
	}
}

func TestModifyLabel(t *testing.T) {
	cfg := cfgRoot(t)
	a := node.New("a", "h", "1", "", nil, nil)
	a.Label = "old"
	b := node.New("b", "h", "1", "", nil, nil)
	b.Label = "taken"
	addParsed(t, cfg, a, b)
	if err := ModifyLabel(cfg, "old", "taken"); err == nil {
		t.Fatal("expected duplicate label error")
	}
	if err := ModifyLabel(cfg, "old", "new"); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.ParsedDir())
	if list.FindByLabel("new") == nil {
		t.Fatal("relabel failed")
	}
}

func TestShowNotFound(t *testing.T) {
	cfg := cfgRoot(t)
	err := Show(cfg, "nope", ShowOptions{})
	if err == nil || !strings.Contains(err.Error(), "No label 'nope' found in list 'parsed'") {
		t.Fatalf("got %v", err)
	}
}

func TestDumpBuffer(t *testing.T) {
	cfg := cfgRoot(t)
	addBuffer(t, cfg, node.New("id1", "h", "1", "", nil, nil))
	if err := DumpBuffer(cfg); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.BufferDir())
	if len(list.Nodes()) != 0 {
		t.Fatal("buffer not empty")
	}
}

func TestSelectInvalidRange(t *testing.T) {
	cfg := cfgRoot(t)
	_, err := selectnode.Resolve(cfg, "node[5-1]", selectnode.Options{})
	if err == nil || !strings.Contains(err.Error(), "InvalidRangeError") {
		t.Fatalf("got %v", err)
	}
}

func TestBufferModifyGroups(t *testing.T) {
	cfg := cfgRoot(t)
	addBuffer(t, cfg, node.New("abc123", "h", "1", "", nil, nil))
	if err := ModifyGroups(cfg, "abc123", ModifyGroupsOptions{Add: "gpu", Buffer: true}); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Load(cfg.BufferDir())
	if !list.Nodes()[0].HasGroup("gpu") {
		t.Fatal("buffer groups not updated")
	}
}
