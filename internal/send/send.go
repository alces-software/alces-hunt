// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package send

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sierra-tango-echo/alces-hunt/internal/collector"
	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/pidfile"
	"github.com/sierra-tango-echo/alces-hunt/internal/protocol"
)

// Options are CLI flags for send.
type Options struct {
	Command          string
	Port             string
	Server           string
	Auth             string
	Broadcast        bool
	BroadcastAddress string
	Groups           []string
	Label            string
	LabelSet         bool
	Prefix           string
	RetryInterval    string
	Sysroot          string
}

var numericRE = regexp.MustCompile(`^\d+(\.\d+)?$`)

// Run transmits this node's identity to a hunt server.
func Run(cfg *config.Config, opt Options) error {
	port := opt.Port
	if port == "" {
		port = cfg.Port
	}
	if port == "" {
		return fmt.Errorf("No port provided!")
	}

	payload, err := prepare(cfg, opt)
	if err != nil {
		return err
	}

	cleanup, err := pidfile.WriteIfConfigured()
	if err != nil {
		return err
	}
	defer cleanup()

	if os.Getenv("ALCES_HUNT_pidfile") != "" {
		time.Sleep(time.Second)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	broadcast := opt.Broadcast || cfg.Broadcast
	if broadcast {
		addr := opt.BroadcastAddress
		if addr == "" {
			addr = cfg.BroadcastAddress
		}
		if addr == "" {
			return fmt.Errorf("No broadcast targets provided!")
		}
		return sendUDP(addr, port, body)
	}

	host := opt.Server
	if host == "" {
		host = cfg.TargetHost
	}
	if host == "" {
		return fmt.Errorf("No target server provided!")
	}
	return sendTCP(host, port, body, retrySeconds(cfg, opt.RetryInterval))
}

func prepare(cfg *config.Config, opt Options) (*protocol.Payload, error) {
	auth := opt.Auth
	if auth == "" {
		auth = cfg.AuthKey
	}

	var label string
	if opt.LabelSet {
		label = opt.Label
	} else {
		serial, err := collector.DefaultSerialLabel()
		if err != nil {
			return nil, err
		}
		label = serial
	}

	prefix := opt.Prefix
	if prefix == "" {
		prefix = cfg.Presets.Prefix
	}

	groups := opt.Groups
	if len(groups) == 0 {
		groups = cfg.Presets.Groups
	}

	var content string
	cmd := opt.Command
	if cmd == "" {
		cmd = cfg.ContentCommand
	}
	if cmd == "" {
		c, err := collector.Collect(opt.Sysroot)
		if err != nil {
			return nil, err
		}
		content = c
	} else {
		c, err := collector.RunCommand(cmd)
		if err != nil {
			return nil, fmt.Errorf("content command failed: %w", err)
		}
		content = c
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &protocol.Payload{
		HostID:   collector.HostID(opt.Sysroot),
		Hostname: hostname,
		Content:  content,
		Label:    label,
		Prefix:   prefix,
		Groups:   groups,
		AuthKey:  auth,
	}, nil
}

func sendUDP(address, port string, body []byte) error {
	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(address, port))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return err
	}
	defer conn.Close()
	if raw, err := conn.SyscallConn(); err == nil {
		_ = raw.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		})
	}
	_, err = conn.WriteToUDP(body, raddr)
	return err
}

func sendTCP(host, port string, body []byte, retry *float64) error {
	url := fmt.Sprintf("http://%s/", net.JoinHostPort(host, port))

	for {
		status, msg, err := doPOST(url, body)
		if err == nil && status == 200 {
			fmt.Println("Successful transmission")
			return nil
		}
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if retry == nil {
			if msg == "" {
				msg = "Unknown HTTP error"
			}
			return fmt.Errorf("%s", msg)
		}
		fmt.Println(msg)
		time.Sleep(time.Duration(*retry * float64(time.Second)))
	}
}

func doPOST(url string, body []byte) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if isConnErr(err) {
			return 0, "The server is unavailable\n" + err.Error(), err
		}
		return 0, err.Error(), err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == 200 {
		return 200, "", nil
	}
	if resp.StatusCode == 401 {
		return 401, "Authentication key mismatch", fmt.Errorf("Authentication key mismatch")
	}
	return resp.StatusCode, "Unknown HTTP error", fmt.Errorf("Unknown HTTP error")
}

func isConnErr(err error) bool {
	s := err.Error()
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "no route to host") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "server misbehaving")
}

func retrySeconds(cfg *config.Config, flag string) *float64 {
	ri := flag
	if ri == "" {
		ri = cfg.RetryInterval
	}
	if strings.TrimSpace(ri) == "" {
		return nil
	}
	if !numericRE.MatchString(ri) {
		v := maxFloat(5.0, toF(ri))
		fmt.Printf("Warning! Invalid value detected for --retry-interval. It has now been set to %s.\n", formatFloat(v))
		return &v
	}
	f := toF(ri)
	if f < 5.0 {
		fmt.Println("Warning! The value for --retry-interval is too small. It has now been set to 5.0.")
		v := 5.0
		return &v
	}
	return &f
}

func toF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
