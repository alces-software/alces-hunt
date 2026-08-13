// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package commands

import (
	"fmt"
	"strings"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/selectnode"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
	"github.com/sierra-tango-echo/alces-hunt/internal/table"
)

// ModifyGroupsOptions are flags for modify-groups.
type ModifyGroupsOptions struct {
	Add    string
	Remove string
	Buffer bool
	Regex  bool
}

// ModifyGroups adds or removes groups on matching nodes.
func ModifyGroups(cfg *config.Config, arg string, opt ModifyGroupsOptions) error {
	if arg == "" {
		return fmt.Errorf("NODE is required")
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	res, err := selectnode.Resolve(cfg, arg, selectnode.Options{
		Buffer: opt.Buffer,
		Regex:  opt.Regex,
	})
	if err != nil {
		return err
	}
	if len(res.Nodes) == 0 {
		return selectnode.NotFoundCollection(res.SearchField, res.ListName, arg)
	}
	toAdd := splitCSV(opt.Add)
	toRemove := splitCSV(opt.Remove)
	for _, n := range res.Nodes {
		n.AddGroups(toAdd)
		n.RemoveGroups(toRemove)
	}
	if err := res.List.Save(); err != nil {
		return err
	}
	fmt.Println("Node(s) updated successfully:")
	fmt.Println(table.FromNodes(res.Nodes, opt.Buffer))
	return nil
}

// RenameGroup renames a group across a list.
func RenameGroup(cfg *config.Config, oldName, newName string, buffer bool) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("GROUP and NEW_NAME are required")
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	dir := cfg.ParsedDir()
	if buffer {
		dir = cfg.BufferDir()
	}
	list, err := store.Load(dir)
	if err != nil {
		return err
	}
	found := false
	for _, n := range list.Nodes() {
		if n.HasGroup(oldName) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Group '%s' does not exist in list '%s'", oldName, list.Name())
	}
	for _, n := range list.Nodes() {
		n.RenameGroup(oldName, newName)
	}
	if err := list.Save(); err != nil {
		return err
	}
	fmt.Printf("Group '%s' in list '%s' renamed to '%s'\n", oldName, list.Name(), newName)
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
