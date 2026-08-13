// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package send

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
)

func TestRetryIntervalTooSmall(t *testing.T) {
	cfg := &config.Config{}
	out := captureStdout(t, func() {
		got := retrySeconds(cfg, "3")
		if got == nil || *got != 5.0 {
			t.Fatalf("got %v", got)
		}
	})
	if !strings.Contains(out, "too small") || !strings.Contains(out, "5.0") {
		t.Fatalf("warning missing: %q", out)
	}
}

func TestRetryIntervalValid(t *testing.T) {
	cfg := &config.Config{}
	got := retrySeconds(cfg, "10")
	if got == nil || *got != 10.0 {
		t.Fatalf("got %v", got)
	}
}

func TestRetryIntervalInvalid(t *testing.T) {
	cfg := &config.Config{}
	out := captureStdout(t, func() {
		got := retrySeconds(cfg, "abc")
		if got == nil || *got != 5.0 {
			t.Fatalf("got %v", got)
		}
	})
	if !strings.Contains(out, "Invalid value") {
		t.Fatalf("warning missing: %q", out)
	}
}

func TestPrepareRequiresDmidecode(t *testing.T) {
	t.Setenv("PATH", t.TempDir()+":/bin:/usr/bin")
	cfg := &config.Config{AuthKey: "k"}
	_, err := prepare(cfg, Options{})
	if err == nil || !strings.Contains(err.Error(), "dmidecode") {
		t.Fatalf("expected dmidecode error, got %v", err)
	}
}

func TestPrepareExplicitLabelSkipsDmidecode(t *testing.T) {
	t.Setenv("PATH", t.TempDir()+":/bin:/usr/bin")
	cfg := &config.Config{AuthKey: "secret"}
	p, err := prepare(cfg, Options{Label: "foobar01", LabelSet: true, Command: "echo hello"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "foobar01" {
		t.Fatalf("label %q", p.Label)
	}
	if p.Content != "hello" {
		t.Fatalf("content %q", p.Content)
	}
	if p.AuthKey != "secret" {
		t.Fatalf("auth %q", p.AuthKey)
	}
}

func TestPrepareDefaultSerial(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dmidecode"), []byte("#!/bin/sh\necho SERIAL99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/bin:/usr/bin")
	cfg := &config.Config{}
	p, err := prepare(cfg, Options{Command: "echo x"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "SERIAL99" {
		t.Fatalf("label %q", p.Label)
	}
}

func TestNoPort(t *testing.T) {
	err := Run(&config.Config{}, Options{LabelSet: true, Label: "x"})
	if err == nil || err.Error() != "No port provided!" {
		t.Fatalf("got %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
