// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package selectnode

import (
	"fmt"
	"regexp"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/genders"
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

// Options control how a NODE[,NODE...] argument is resolved.
type Options struct {
	Buffer        bool
	Regex         bool
	MatchHostname bool
}

// Result is a resolved selection against one list.
type Result struct {
	List        *store.List
	Nodes       []*node.Node
	SearchField string
	ListName    string
}

// Resolve expands the argument and finds matching nodes.
func Resolve(cfg *config.Config, arg string, opt Options) (*Result, error) {
	dir := cfg.ParsedDir()
	field := "label"
	if opt.Buffer {
		dir = cfg.BufferDir()
		field = "id"
	}
	if opt.MatchHostname {
		field = "hostname"
	}
	list, err := store.Load(dir)
	if err != nil {
		return nil, err
	}
	var patterns []string
	if opt.Regex {
		patterns = genders.SplitRegex(arg)
	} else {
		patterns, err = genders.Expand(arg)
		if err != nil {
			return nil, err
		}
	}
	var found []*node.Node
	seen := map[string]struct{}{}
	if opt.Regex {
		for _, p := range patterns {
			re, err := regexp.Compile(config.NormalizeRegex(p))
			if err != nil {
				return nil, fmt.Errorf("invalid regular expression %q: %w", p, err)
			}
			for _, n := range list.Nodes() {
				if re.MatchString(fieldValue(n, field)) {
					if _, ok := seen[n.ID]; ok {
						continue
					}
					seen[n.ID] = struct{}{}
					found = append(found, n)
				}
			}
		}
	} else {
		for _, p := range patterns {
			for _, n := range list.Nodes() {
				if fieldValue(n, field) == p {
					if _, ok := seen[n.ID]; ok {
						continue
					}
					seen[n.ID] = struct{}{}
					found = append(found, n)
				}
			}
		}
	}
	return &Result{
		List:        list,
		Nodes:       found,
		SearchField: field,
		ListName:    list.Name(),
	}, nil
}

func fieldValue(n *node.Node, field string) string {
	switch field {
	case "id":
		return n.ID
	case "hostname":
		return n.Hostname
	default:
		return n.Label
	}
}

// NotFoundCollection is the remove/modify-groups empty-match error.
func NotFoundCollection(field, listName, arg string) error {
	return fmt.Errorf("No %ss in list '%s' found in collection '%s'", field, listName, arg)
}

// NotFoundOne is the show empty-match error.
func NotFoundOne(field, value, listName string) error {
	return fmt.Errorf("No %s '%s' found in list '%s'", field, value, listName)
}
