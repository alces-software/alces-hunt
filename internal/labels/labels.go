// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package labels

import (
	"strconv"
	"strings"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

// Options control automatic label generation.
type Options struct {
	Prefix       string
	Start        string
	DefaultLabel string
	DefaultStart string
	PrefixStarts map[string]string
}

// HostnameLabel applies default_label to a hostname.
func HostnameLabel(hostname, defaultLabel string) string {
	switch strings.ToLower(defaultLabel) {
	case "short":
		if i := strings.IndexByte(hostname, '.'); i >= 0 {
			return hostname[:i]
		}
		return hostname
	case "blank":
		return ""
	default:
		return hostname
	}
}

// AutoLabel generates a label from prefix + start, or falls back to hostname.
//
// Padding width = max(0, len(start) - len(strconv.Itoa(i))).
// While the candidate is already in used, the counter is incremented.
func AutoLabel(n *node.Node, used []string, opt Options) string {
	prefix := ""
	if n.Presets != nil {
		prefix = n.Presets["prefix"]
	}
	if prefix == "" {
		prefix = opt.Prefix
	}
	hostname := HostnameLabel(n.Hostname, opt.DefaultLabel)
	if prefix == "" {
		return hostname
	}

	start := ""
	if opt.PrefixStarts != nil {
		start = opt.PrefixStarts[prefix]
	}
	if start == "" {
		start = opt.Start
	}
	if start == "" {
		start = opt.DefaultStart
	}
	if start == "" {
		start = "01"
	}

	i, _ := strconv.Atoi(start)
	usedSet := map[string]struct{}{}
	for _, u := range used {
		usedSet[u] = struct{}{}
	}
	for {
		num := strconv.Itoa(i)
		pad := len(start) - len(num)
		if pad < 0 {
			pad = 0
		}
		name := prefix + strings.Repeat("0", pad) + num
		if _, ok := usedSet[name]; !ok {
			return name
		}
		i++
	}
}

// Contains reports whether name is in the list.
func Contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// FirstDuplicate returns the first value that appears more than once.
func FirstDuplicate(names []string) string {
	seen := map[string]int{}
	for _, n := range names {
		seen[n]++
	}
	for _, n := range names {
		if seen[n] > 1 {
			return n
		}
	}
	return ""
}
