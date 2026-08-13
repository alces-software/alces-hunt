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

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/output"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
	"github.com/sierra-tango-echo/alces-hunt/internal/table"
)

// ListOptions are flags for list.
type ListOptions struct {
	Plain   bool
	ByGroup bool
	Buffer  bool
}

// List prints the buffer or parsed node list.
func List(cfg *config.Config, opt ListOptions) error {
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	dir := cfg.ParsedDir()
	if opt.Buffer {
		dir = cfg.BufferDir()
	}
	list, err := store.Load(dir)
	if err != nil {
		return err
	}
	nodes := list.Nodes()
	if opt.Plain {
		// --by-group is ignored in plain mode.
		fmt.Print(output.ListPlainAll(nodes))
		return nil
	}
	if len(nodes) == 0 {
		return fmt.Errorf("No nodes to display")
	}
	if opt.ByGroup {
		groups, names := list.ByGroup()
		for _, name := range names {
			fmt.Printf("Group '%s':\n", name)
			fmt.Println(table.FromNodes(groups[name], opt.Buffer))
		}
		return nil
	}
	fmt.Println(table.FromNodes(nodes, opt.Buffer))
	return nil
}
