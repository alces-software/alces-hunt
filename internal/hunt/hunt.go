// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package hunt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/labels"
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/pidfile"
	"github.com/sierra-tango-echo/alces-hunt/internal/profile"
	"github.com/sierra-tango-echo/alces-hunt/internal/protocol"
	"github.com/sierra-tango-echo/alces-hunt/internal/send"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

// Options are CLI flags for hunt.
type Options struct {
	Port          string
	AllowExisting bool
	IncludeSelf   bool
	Auth          string
	AutoParse     string
}

// Run starts the dual TCP+UDP listener.
func Run(cfg *config.Config, opt Options) error {
	port := opt.Port
	if port == "" {
		port = cfg.Port
	}
	if port == "" {
		return fmt.Errorf("No port provided!")
	}
	if busy, err := portBusy(port); err != nil {
		return err
	} else if busy {
		return fmt.Errorf("Provided port %s is busy", port)
	}

	autoParse := opt.AutoParse
	if autoParse == "" {
		autoParse = cfg.AutoParse
	}
	if autoParse == "" {
		autoParse = ".^"
	}
	re, err := config.CompileRegex(autoParse)
	if err != nil {
		return fmt.Errorf("Invalid regular expression passed to `auto_parse` option")
	}

	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	cleanup, err := pidfile.WriteIfConfigured()
	if err != nil {
		return err
	}
	defer cleanup()

	auth := opt.Auth
	if auth == "" {
		auth = cfg.AuthKey
	}

	srv := &server{
		cfg:           cfg,
		port:          port,
		auth:          auth,
		autoParse:     re,
		allowExisting: opt.AllowExisting || cfg.AllowExisting,
	}

	tcpLn, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if isBusy(err) {
			return fmt.Errorf("Provided port %s is busy", port)
		}
		return err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		tcpLn.Close()
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		tcpLn.Close()
		if isBusy(err) {
			return fmt.Errorf("Provided port %s is busy", port)
		}
		return err
	}

	fmt.Printf("Hunter running on port %s - Ctrl+C to stop\n", port)

	errCh := make(chan error, 2)
	go func() { errCh <- srv.serveTCP(tcpLn) }()
	go func() { errCh <- srv.serveUDP(udpConn) }()

	if opt.IncludeSelf || cfg.IncludeSelf {
		host := cfg.TargetHost
		if host == "" {
			host = "localhost"
		}
		// include-self uses the same send path (including dmidecode default label).
		_ = os.Unsetenv("ALCES_HUNT_pidfile")
		if err := send.Run(cfg, send.Options{
			Port:   port,
			Server: host,
			Auth:   auth,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "include-self send failed: %s\n", err)
		}
	}

	return <-errCh
}

type server struct {
	cfg           *config.Config
	port          string
	auth          string
	autoParse     interface{ MatchString(string) bool }
	allowExisting bool
	mu            sync.Mutex
}

func (s *server) serveTCP(ln net.Listener) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleTCP(conn)
	}
}

func (s *server) handleTCP(conn net.Conn) {
	defer conn.Close()
	peer := peerIP(conn.RemoteAddr())
	br := bufio.NewReader(conn)
	headers := map[string]string{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if len(headers) == 0 {
				return
			}
			fmt.Fprintf(conn, "HTTP/1.1 500\r\n\r\n")
			fmt.Println("Caught exception: unknown nil line captured from the client socket")
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		// Ignore probe connections that send no request line.
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	if headers["Content-Type"] != "application/json" {
		fmt.Printf("Malformed packet received from %s\n", peer)
		fmt.Fprintf(conn, "HTTP/1.1 415\r\n\r\n")
		return
	}
	n, _ := strconv.Atoi(headers["Content-Length"])
	body := make([]byte, n)
	if _, err := io.ReadFull(br, body); err != nil {
		fmt.Printf("Malformed packet received from %s\n", peer)
		fmt.Fprintf(conn, "HTTP/1.1 400\r\n\r\n")
		return
	}
	var payload protocol.Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Printf("Malformed packet received from %s\n", peer)
		fmt.Fprintf(conn, "HTTP/1.1 400\r\n\r\n")
		return
	}
	if payload.AuthKey != s.auth {
		fmt.Fprintf(conn, "HTTP/1.1 401\r\n\r\n")
		fmt.Println("Unauthorised node attempted to connect")
		return
	}
	fmt.Fprintf(conn, "HTTP/1.1 200\r\n\r\n")
	s.process(payload, peer)
}

func (s *server) serveUDP(conn *net.UDPConn) error {
	defer conn.Close()
	buf := make([]byte, 65535)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		ip := addr.IP.String()
		if !json.Valid(buf[:n]) {
			fmt.Printf("Malformed packet received from %s\n", ip)
			continue
		}
		var payload protocol.Payload
		if err := json.Unmarshal(buf[:n], &payload); err != nil {
			fmt.Printf("Malformed packet received from %s\n", ip)
			continue
		}
		if payload.AuthKey != s.auth {
			fmt.Println("Unauthorised node attempted to connect")
			continue
		}
		s.process(payload, ip)
	}
}

func (s *server) process(data protocol.Payload, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := node.New(data.HostID, data.Hostname, ip, data.Content, data.Groups, map[string]string{
		"label":  data.Label,
		"prefix": data.Prefix,
	})

	fmt.Printf("Found node.\nID: %s\nName: %s\nIP: %s\n\n", n.ID, n.Hostname, n.IP)

	buffer, err := store.Load(s.cfg.BufferDir())
	if err != nil {
		fmt.Println(err)
		return
	}
	parsed, err := store.Load(s.cfg.ParsedDir())
	if err != nil {
		fmt.Println(err)
		return
	}

	dest := buffer
	if s.autoParse.MatchString(n.Hostname) {
		dest = parsed
		n.Label = n.PresetLabel()
		if n.Label == "" {
			n.Label = labels.AutoLabel(n, parsed.Labels(), labels.Options{
				DefaultLabel: s.cfg.DefaultLabel,
				DefaultStart: s.cfg.DefaultStart,
				PrefixStarts: s.cfg.PrefixStarts,
			})
		}
		if parsed.IncludeLabel(n.Label) {
			fmt.Printf("Node %s could not be auto-parsed as the resolved name matches an existing node\n", n.Hostname)
			return
		}
	}

	if s.allowExisting {
		_ = buffer.DeleteByID(n.ID)
		_ = parsed.DeleteByID(n.ID)
		n.AutoApply = dest.Name() == "parsed"
		dest.Add(n)
		fmt.Printf("Node added to %s node list\n", dest.Name())
	} else if buffer.IncludeID(n.ID) {
		fmt.Println("ID already exists in buffer")
	} else if parsed.IncludeID(n.ID) {
		fmt.Println("ID already exists in parsed node list")
	} else {
		n.AutoApply = dest.Name() == "parsed"
		dest.Add(n)
		fmt.Printf("Node added to %s node list\n", dest.Name())
	}

	if err := dest.Save(); err != nil {
		fmt.Println(err)
		return
	}
	if n.AutoApply {
		if err := profile.ApplyRules(s.cfg, n.Label); err != nil {
			fmt.Printf("ERROR: %s\n", err)
		}
	}
}

func portBusy(port string) (bool, error) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if isBusy(err) {
			return true, nil
		}
		return false, err
	}
	ln.Close()
	return false, nil
}

func isBusy(err error) bool {
	if op, ok := err.(*net.OpError); ok {
		return strings.Contains(op.Err.Error(), "address already in use")
	}
	return strings.Contains(err.Error(), "address already in use")
}

func peerIP(addr net.Addr) string {
	if addr == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
