// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverridesPort(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "config.yml"), []byte("port: 1111\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALCES_HUNT_ROOT", root)
	t.Setenv("ALCES_HUNT_port", "1234")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "1234" {
		t.Fatalf("port=%q", cfg.Port)
	}
}

func TestAutoApplyInvalidRegex(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	yml := "auto_apply:\n  \"[unterminated\": ident\n"
	if err := os.WriteFile(filepath.Join(root, "etc", "config.yml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALCES_HUNT_ROOT", root)
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestAutoApplyEnvJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ALCES_HUNT_ROOT", root)
	t.Setenv("ALCES_HUNT_auto_apply", `{"^cnode":"compute"}`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AutoApply) != 1 || cfg.AutoApply[0].Identity != "compute" {
		t.Fatalf("auto_apply=%v", cfg.AutoApply)
	}
}

func TestNormalizeRegex(t *testing.T) {
	if NormalizeRegex("/^cnode/") != "^cnode" {
		t.Fatal(NormalizeRegex("/^cnode/"))
	}
	if NormalizeRegex("^cnode") != "^cnode" {
		t.Fatal("unchanged")
	}
}

func TestDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ALCES_HUNT_ROOT", root)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultLabel != "long" || cfg.DefaultStart != "01" {
		t.Fatalf("defaults %+v", cfg)
	}
	if cfg.AutoParse != ".^" {
		t.Fatalf("auto_parse default %q", cfg.AutoParse)
	}
}
