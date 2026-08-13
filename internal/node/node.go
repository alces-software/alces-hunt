// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package node

import (
	"encoding/json"
	"sort"
	"strings"
)

// Node is the persisted inventory record.
type Node struct {
	ID        string            `yaml:"id" json:"id"`
	Hostname  string            `yaml:"hostname" json:"hostname"`
	Label     string            `yaml:"label,omitempty" json:"label,omitempty"`
	IP        string            `yaml:"ip" json:"ip"`
	Content   string            `yaml:"content" json:"content"`
	Groups    []string          `yaml:"groups" json:"groups"`
	Presets   map[string]string `yaml:"presets" json:"presets"`
	AutoApply bool              `yaml:"-" json:"-"`
}

// New constructs a node with empty-safe groups and presets.
func New(id, hostname, ip, content string, groups []string, presets map[string]string) *Node {
	n := &Node{
		ID:       id,
		Hostname: hostname,
		IP:       ip,
		Content:  content,
		Groups:   []string{},
		Presets:  map[string]string{},
	}
	n.AddGroups(groups)
	n.SetPresets(presets)
	return n
}

// SetPresets stores non-empty string presets.
func (n *Node) SetPresets(presets map[string]string) {
	n.Presets = map[string]string{}
	for k, v := range presets {
		if strings.TrimSpace(v) != "" {
			n.Presets[k] = v
		}
	}
}

// SortedGroups returns groups in sorted order.
func (n *Node) SortedGroups() []string {
	out := append([]string{}, n.Groups...)
	sort.Strings(out)
	return out
}

// PrettyGroups is a comma-separated sorted group list.
func (n *Node) PrettyGroups() string {
	return strings.Join(n.SortedGroups(), ", ")
}

// PrettyPresets is a multi-line key: 'value' rendering.
func (n *Node) PrettyPresets() string {
	if len(n.Presets) == 0 {
		return ""
	}
	keys := make([]string, 0, len(n.Presets))
	for k := range n.Presets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteString(": '")
		b.WriteString(n.Presets[k])
		b.WriteByte('\'')
	}
	return b.String()
}

// CompactPresetsJSON is compact JSON for --plain list output.
func (n *Node) CompactPresetsJSON() string {
	if n.Presets == nil {
		return "{}"
	}
	b, err := json.Marshal(n.Presets)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// PlainGroups is the --plain groups field: g1|g2 or |.
func (n *Node) PlainGroups() string {
	g := n.SortedGroups()
	if len(g) == 0 {
		return "|"
	}
	return strings.Join(g, "|")
}

// AddGroups appends unique group names.
func (n *Node) AddGroups(groups []string) {
	seen := map[string]struct{}{}
	for _, g := range n.Groups {
		seen[g] = struct{}{}
	}
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if _, ok := seen[g]; ok {
			continue
		}
		n.Groups = append(n.Groups, g)
		seen[g] = struct{}{}
	}
}

// RemoveGroups drops the named groups.
func (n *Node) RemoveGroups(groups []string) {
	drop := map[string]struct{}{}
	for _, g := range groups {
		drop[strings.TrimSpace(g)] = struct{}{}
	}
	out := n.Groups[:0]
	for _, g := range n.Groups {
		if _, ok := drop[g]; !ok {
			out = append(out, g)
		}
	}
	n.Groups = out
}

// RenameGroup replaces old with new across this node's groups.
func (n *Node) RenameGroup(old, newName string) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(n.Groups))
	for _, g := range n.Groups {
		if g == old {
			g = newName
		}
		if _, ok := seen[g]; ok {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	n.Groups = out
}

// HasGroup reports whether the node belongs to group.
func (n *Node) HasGroup(group string) bool {
	for _, g := range n.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// PresetLabel is the already-assigned label or the client preset label.
func (n *Node) PresetLabel() string {
	if n.Label != "" {
		return n.Label
	}
	if n.Presets != nil {
		return n.Presets["label"]
	}
	return ""
}

// Clone returns a shallow copy with independent groups and presets maps.
func (n *Node) Clone() *Node {
	c := *n
	c.Groups = append([]string{}, n.Groups...)
	c.Presets = map[string]string{}
	for k, v := range n.Presets {
		c.Presets[k] = v
	}
	return &c
}
