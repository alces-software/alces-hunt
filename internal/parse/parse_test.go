// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package parse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

func testCfg(t *testing.T) *config.Config {
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

func seedBuffer(t *testing.T, cfg *config.Config, nodes ...*node.Node) {
	t.Helper()
	buf, err := store.Load(cfg.BufferDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		buf.Add(n)
	}
	if err := buf.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoPrefixSequence(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg,
		node.New("id1", "h1", "10.0.0.1", "", nil, nil),
		node.New("id2", "h2", "10.0.0.2", "", nil, nil),
		node.New("id3", "h3", "10.0.0.3", "", nil, nil),
	)
	if err := Run(cfg, Options{Auto: true, Prefix: "cnode", Start: "001"}); err != nil {
		t.Fatal(err)
	}
	parsed, _ := store.Load(cfg.ParsedDir())
	nodes := parsed.Nodes()
	if len(nodes) != 3 {
		t.Fatalf("parsed %d", len(nodes))
	}
	// Nodes are sorted by id: id1, id2, id3 — generation followed buffer order which
	// after reload is also sorted by id.
	want := []string{"cnode001", "cnode002", "cnode003"}
	for i, n := range nodes {
		if n.Label != want[i] {
			t.Errorf("node %s label %q want %q", n.ID, n.Label, want[i])
		}
	}
	buf, _ := store.Load(cfg.BufferDir())
	if len(buf.Nodes()) != 0 {
		t.Fatal("buffer should be empty")
	}
}

func TestAutoPresetLabel(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "h1", "10.0.0.1", "", nil, map[string]string{"label": "foobar01"}))
	if err := Run(cfg, Options{Auto: true, Prefix: "cnode", Start: "001"}); err != nil {
		t.Fatal(err)
	}
	parsed, _ := store.Load(cfg.ParsedDir())
	if parsed.Nodes()[0].Label != "foobar01" {
		t.Fatalf("preset should win, got %q", parsed.Nodes()[0].Label)
	}
}

func TestAutoDuplicatePresetFails(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg,
		node.New("id1", "h1", "10.0.0.1", "", nil, map[string]string{"label": "same"}),
		node.New("id2", "h2", "10.0.0.2", "", nil, map[string]string{"label": "same"}),
	)
	err := Run(cfg, Options{Auto: true})
	if err == nil {
		t.Fatal("expected duplicate preset error")
	}
	parsed, _ := store.Load(cfg.ParsedDir())
	if len(parsed.Nodes()) != 0 {
		t.Fatal("nothing should have been written")
	}
}

func TestAutoLabelCollisionWithoutSkip(t *testing.T) {
	cfg := testCfg(t)
	parsed, _ := store.Load(cfg.ParsedDir())
	existing := node.New("old", "hx", "9.9.9.9", "", nil, nil)
	existing.Label = "cnode001"
	parsed.Add(existing)
	_ = parsed.Save()
	seedBuffer(t, cfg, node.New("id1", "h1", "10.0.0.1", "", nil, nil))
	err := Run(cfg, Options{Auto: true, Prefix: "cnode", Start: "001"})
	if err == nil {
		t.Fatal("expected collision error")
	}
}

func TestAutoSkipUsedIndex(t *testing.T) {
	cfg := testCfg(t)
	parsed, _ := store.Load(cfg.ParsedDir())
	existing := node.New("old", "hx", "9.9.9.9", "", nil, nil)
	existing.Label = "cnode001"
	parsed.Add(existing)
	_ = parsed.Save()
	seedBuffer(t, cfg, node.New("id1", "h1", "10.0.0.1", "", nil, nil))
	if err := Run(cfg, Options{Auto: true, Prefix: "cnode", Start: "001", SkipSet: true, SkipUsedIndex: true}); err != nil {
		t.Fatal(err)
	}
	parsed, _ = store.Load(cfg.ParsedDir())
	n := parsed.FindByID("id1")
	if n == nil || n.Label != "cnode002" {
		t.Fatalf("want cnode002, got %+v", n)
	}
}

func TestDryRunDoesNotMove(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "h1", "10.0.0.1", "", nil, nil))
	if err := Run(cfg, Options{Auto: true, Prefix: "cnode", Start: "001", DryRun: true}); err != nil {
		t.Fatal(err)
	}
	buf, _ := store.Load(cfg.BufferDir())
	parsed, _ := store.Load(cfg.ParsedDir())
	if len(buf.Nodes()) != 1 || len(parsed.Nodes()) != 0 {
		t.Fatal("dry-run must not move nodes")
	}
}

func TestDefaultLabelLongShortBlank(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "login.example.com", "10.0.0.1", "", nil, nil))
	if err := Run(cfg, Options{Auto: true, DefaultLabel: "short"}); err != nil {
		t.Fatal(err)
	}
	parsed, _ := store.Load(cfg.ParsedDir())
	if parsed.Nodes()[0].Label != "login" {
		t.Fatalf("got %q", parsed.Nodes()[0].Label)
	}

	cfg = testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "login.example.com", "10.0.0.1", "", nil, nil))
	err := Run(cfg, Options{Auto: true, DefaultLabel: "blank"})
	if err == nil {
		t.Fatal("blank label should be rejected")
	}
}

func TestAllowExistingReplaces(t *testing.T) {
	cfg := testCfg(t)
	parsed, _ := store.Load(cfg.ParsedDir())
	old := node.New("id1", "oldhost", "1.1.1.1", "old", nil, nil)
	old.Label = "oldlabel"
	parsed.Add(old)
	_ = parsed.Save()
	seedBuffer(t, cfg, node.New("id1", "newhost", "2.2.2.2", "new", nil, map[string]string{"label": "newlabel"}))
	if err := Run(cfg, Options{Auto: true, AllowExisting: true}); err != nil {
		t.Fatal(err)
	}
	parsed, _ = store.Load(cfg.ParsedDir())
	if len(parsed.Nodes()) != 1 || parsed.Nodes()[0].Label != "newlabel" {
		t.Fatalf("replace failed: %+v", parsed.Nodes())
	}
}

func TestAutoApplyBestEffort(t *testing.T) {
	cfg := testCfg(t)
	cfg.AutoApply = []config.ApplyRule{{Regex: ".*", Identity: "compute"}}
	cfg.ProfileCommand = []string{filepath.Join(cfg.Root, "no-such-binary")}
	seedBuffer(t, cfg, node.New("id1", "h1", "10.0.0.1", "", nil, map[string]string{"label": "n01"}))
	if err := Run(cfg, Options{Auto: true}); err != nil {
		t.Fatal(err)
	}
	parsed, _ := store.Load(cfg.ParsedDir())
	if len(parsed.Nodes()) != 1 {
		t.Fatal("parse should succeed even if profile command is missing")
	}
}

func TestEmptyBufferError(t *testing.T) {
	cfg := testCfg(t)
	if err := Run(cfg, Options{Auto: true}); err == nil {
		t.Fatal("expected empty buffer error")
	}
}

func TestInvalidDefaultLabel(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "h", "1.1.1.1", "", nil, nil))
	if err := Run(cfg, Options{Auto: true, DefaultLabel: "nope"}); err == nil {
		t.Fatal("expected invalid default_label")
	}
}

func TestDumpLeavesNothing(t *testing.T) {
	cfg := testCfg(t)
	seedBuffer(t, cfg, node.New("id1", "h", "1.1.1.1", "", nil, nil))
	buf, _ := store.Load(cfg.BufferDir())
	_ = buf.Empty()
	entries, _ := os.ReadDir(cfg.BufferDir())
	if len(entries) != 0 {
		t.Fatalf("buffer dir not empty: %v", entries)
	}
}
