// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectFromSysroot(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "proc", "cmdline"), "BOOT_IMAGE=/vmlinuz SYSUUID=abc123 BOOTIF=01-00-11-22-33-44-55\n")
	mustWrite(t, filepath.Join(root, "sys", "class", "net", "eno1", "address"), "00:11:22:33:44:55\n")
	mustWrite(t, filepath.Join(root, "sys", "class", "net", "lo", "address"), "00:00:00:00:00:00\n")
	mustWrite(t, filepath.Join(root, "sys", "class", "block", "sda", "device", "dummy"), "")
	mustWrite(t, filepath.Join(root, "sys", "class", "block", "sda", "size"), "1953525168\n")
	mustWrite(t, filepath.Join(root, "sys", "class", "block", "sda1", "size"), "100\n") // no device/ — skipped

	yml, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(yml, "sysuuid: abc123") {
		t.Fatalf("missing sysuuid: %s", yml)
	}
	if !strings.Contains(yml, "bootif: 01-00-11-22-33-44-55") {
		t.Fatalf("missing bootif: %s", yml)
	}
	if !strings.Contains(yml, "eno1:") || !strings.Contains(yml, "00:11:22:33:44:55") {
		t.Fatalf("missing net: %s", yml)
	}
	if strings.Contains(yml, "lo:") {
		t.Fatalf("loopback should be skipped: %s", yml)
	}
	if !strings.Contains(yml, "sda:") {
		t.Fatalf("missing disk: %s", yml)
	}
	if strings.Contains(yml, "sda1:") {
		t.Fatalf("partition without device/ should be skipped: %s", yml)
	}
}

func TestHostIDFromCmdline(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "proc", "cmdline"), "SYSUUID=deadbeef quiet\n")
	if got := HostID(root); got != "deadbeef" {
		t.Fatalf("hostid=%q", got)
	}
}

func TestDefaultSerialLabelMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()+":/bin:/usr/bin")
	_, err := DefaultSerialLabel()
	if err == nil {
		t.Fatal("expected missing dmidecode error")
	}
	if !strings.Contains(err.Error(), "dmidecode") {
		t.Fatalf("error should mention dmidecode: %v", err)
	}
}

func TestDefaultSerialLabelPresent(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "dmidecode")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho '  SERIAL42  '\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/bin:/usr/bin")
	got, err := DefaultSerialLabel()
	if err != nil {
		t.Fatal(err)
	}
	if got != "SERIAL42" {
		t.Fatalf("got %q", got)
	}
}

func TestRunCommandChomp(t *testing.T) {
	got, err := RunCommand("printf 'hello\\n'")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
