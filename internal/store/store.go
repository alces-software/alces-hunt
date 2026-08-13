// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"gopkg.in/yaml.v3"
)

// List is a file-backed node list (buffer or parsed).
type List struct {
	mu    sync.Mutex
	Dir   string
	nodes []*node.Node
}

// Load reads every YAML file in dir.
func Load(dir string) (*List, error) {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("Node directory %s doesn't exist", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	l := &List{Dir: dir}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var n node.Node
		if err := yaml.Unmarshal(data, &n); err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		if n.Groups == nil {
			n.Groups = []string{}
		}
		if n.Presets == nil {
			n.Presets = map[string]string{}
		}
		l.nodes = append(l.nodes, &n)
	}
	return l, nil
}

// Name is the last path component (buffer or parsed).
func (l *List) Name() string {
	return filepath.Base(l.Dir)
}

// Nodes returns nodes sorted by id.
func (l *List) Nodes() []*node.Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := append([]*node.Node{}, l.nodes...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ByGroup returns a map of group name to member nodes, plus sorted group names.
func (l *List) ByGroup() (map[string][]*node.Node, []string) {
	nodes := l.Nodes()
	groups := map[string][]*node.Node{}
	var names []string
	seen := map[string]struct{}{}
	for _, n := range nodes {
		for _, g := range n.SortedGroups() {
			if _, ok := seen[g]; !ok {
				names = append(names, g)
				seen[g] = struct{}{}
			}
			groups[g] = append(groups[g], n)
		}
	}
	sort.Strings(names)
	return groups, names
}

// IncludeID reports whether a node with id exists.
func (l *List) IncludeID(id string) bool {
	return l.FindByID(id) != nil
}

// IncludeLabel reports whether a node with label exists.
func (l *List) IncludeLabel(label string) bool {
	return l.FindByLabel(label) != nil
}

// FindByID returns the node with the given id.
func (l *List) FindByID(id string) *node.Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range l.nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// FindByLabel returns the node with the given label.
func (l *List) FindByLabel(label string) *node.Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range l.nodes {
		if n.Label == label {
			return n
		}
	}
	return nil
}

// FindByHostname returns the first node with the given hostname.
func (l *List) FindByHostname(hostname string) *node.Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range l.nodes {
		if n.Hostname == hostname {
			return n
		}
	}
	return nil
}

// Add appends a node. The caller is responsible for uniqueness.
func (l *List) Add(n *node.Node) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nodes = append(l.nodes, n)
}

// Delete removes the given nodes from memory and deletes their files.
func (l *List) Delete(nodes []*node.Node) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	drop := map[string]struct{}{}
	for _, n := range nodes {
		drop[n.ID] = struct{}{}
		path := filepath.Join(l.Dir, n.ID+".yaml")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	out := l.nodes[:0]
	for _, n := range l.nodes {
		if _, ok := drop[n.ID]; !ok {
			out = append(out, n)
		}
	}
	l.nodes = out
	return nil
}

// DeleteByID removes a node by id if present.
func (l *List) DeleteByID(id string) error {
	if n := l.FindByID(id); n != nil {
		return l.Delete([]*node.Node{n})
	}
	return nil
}

// Empty deletes every node.
func (l *List) Empty() error {
	return l.Delete(l.Nodes())
}

// ReplaceByID removes any existing record with the same id then adds n.
func (l *List) ReplaceByID(n *node.Node) error {
	if err := l.DeleteByID(n.ID); err != nil {
		return err
	}
	l.Add(n)
	return nil
}

// Save writes every node as <id>.yaml using an atomic rename.
func (l *List) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		return err
	}
	for _, n := range l.nodes {
		n.Groups = n.SortedGroups()
		if err := writeNode(l.Dir, n); err != nil {
			return err
		}
	}
	return nil
}

func writeNode(dir string, n *node.Node) error {
	data, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, n.ID+".yaml")
	tmp, err := os.CreateTemp(dir, ".tmp-"+n.ID+"-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// Labels returns all non-empty labels.
func (l *List) Labels() []string {
	var out []string
	for _, n := range l.Nodes() {
		if n.Label != "" {
			out = append(out, n.Label)
		}
	}
	return out
}
