// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package hunt_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/hunt"
	"github.com/sierra-tango-echo/alces-hunt/internal/send"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

func startHunt(t *testing.T, cfg *config.Config, opt hunt.Options) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- hunt.Run(cfg, opt) }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+opt.Port, 50*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		select {
		case err := <-errCh:
			t.Fatalf("hunt exited: %v", err)
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
	t.Fatal("hunt did not start")
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	ln.Close()
	return port
}

func huntCfg(t *testing.T) *config.Config {
	t.Helper()
	root := t.TempDir()
	t.Setenv("ALCES_HUNT_ROOT", root)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.AuthKey = "secret"
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func fakeDMI(t *testing.T, serial string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dmidecode"), []byte("#!/bin/sh\necho '"+serial+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/bin:/usr/bin:/usr/local/bin")
}

func TestHuntTCPSendAndAuth(t *testing.T) {
	cfg := huntCfg(t)
	port := freePort(t)
	startHunt(t, cfg, hunt.Options{Port: port, Auth: "secret"})
	fakeDMI(t, "SER1")

	if err := send.Run(cfg, send.Options{
		Port:     port,
		Server:   "127.0.0.1",
		Auth:     "secret",
		Label:    "n01",
		LabelSet: true,
		Command:  "echo hello-content",
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	buf, err := store.Load(cfg.BufferDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(buf.Nodes()) != 1 {
		t.Fatalf("want 1 node, got %d", len(buf.Nodes()))
	}
	n := buf.Nodes()[0]
	if n.Hostname == "" || n.IP == "" || n.Content != "hello-content" {
		t.Fatalf("node incomplete: %+v", n)
	}
	if n.Presets["label"] != "n01" {
		t.Fatalf("preset label %v", n.Presets)
	}

	// Duplicate ID without allow-existing
	if err := send.Run(cfg, send.Options{
		Port: port, Server: "127.0.0.1", Auth: "secret",
		LabelSet: true, Label: "n01", Command: "echo again",
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	buf, _ = store.Load(cfg.BufferDir())
	if len(buf.Nodes()) != 1 {
		t.Fatalf("duplicate should be ignored, got %d", len(buf.Nodes()))
	}

	// Auth mismatch
	err = send.Run(cfg, send.Options{
		Port: port, Server: "127.0.0.1", Auth: "wrong",
		LabelSet: true, Label: "x", Command: "echo nope",
	})
	if err == nil || !strings.Contains(err.Error(), "Authentication") {
		t.Fatalf("want auth error, got %v", err)
	}
}

func TestHuntBroadcastUDP(t *testing.T) {
	cfg := huntCfg(t)
	port := freePort(t)
	startHunt(t, cfg, hunt.Options{Port: port, Auth: "secret"})
	if err := send.Run(cfg, send.Options{
		Port:             port,
		Broadcast:        true,
		BroadcastAddress: "127.0.0.1",
		Auth:             "secret",
		LabelSet:         true,
		Label:            "udp1",
		Command:          "echo udp-content",
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		buf, _ := store.Load(cfg.BufferDir())
		if len(buf.Nodes()) == 1 && buf.Nodes()[0].Content == "udp-content" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("udp node not received")
}

func TestHuntAutoParseAny(t *testing.T) {
	cfg := huntCfg(t)
	port := freePort(t)
	startHunt(t, cfg, hunt.Options{Port: port, Auth: "secret", AutoParse: ".*"})
	if err := send.Run(cfg, send.Options{
		Port: port, Server: "127.0.0.1", Auth: "secret",
		LabelSet: true, Label: "auto01", Command: "echo z",
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		parsed, _ := store.Load(cfg.ParsedDir())
		if len(parsed.Nodes()) == 1 && parsed.Nodes()[0].Label == "auto01" {
			buf, _ := store.Load(cfg.BufferDir())
			if len(buf.Nodes()) != 0 {
				t.Fatal("auto-parsed node should not remain in buffer")
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("node was not auto-parsed")
}

func TestHuntPortBusy(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	cfg := huntCfg(t)
	err = hunt.Run(cfg, hunt.Options{Port: port})
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("want busy error, got %v", err)
	}
}

func TestHuntNoPort(t *testing.T) {
	err := hunt.Run(&config.Config{}, hunt.Options{})
	if err == nil || err.Error() != "No port provided!" {
		t.Fatalf("got %v", err)
	}
}

func TestPIDFile(t *testing.T) {
	cfg := huntCfg(t)
	port := freePort(t)
	pid := filepath.Join(t.TempDir(), "hunt.pid")
	t.Setenv("ALCES_HUNT_pidfile", pid)
	startHunt(t, cfg, hunt.Options{Port: port, Auth: "secret"})
	time.Sleep(50 * time.Millisecond)
	data, err := os.ReadFile(pid)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) == "" {
		t.Fatal("empty pidfile")
	}
}
