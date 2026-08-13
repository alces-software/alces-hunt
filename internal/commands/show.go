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
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/output"
	"github.com/sierra-tango-echo/alces-hunt/internal/selectnode"
	"github.com/sierra-tango-echo/alces-hunt/internal/table"
)

// ShowOptions are flags for show.
type ShowOptions struct {
	Buffer bool
	Plain  bool
}

// Show prints details of one node.
func Show(cfg *config.Config, arg string, opt ShowOptions) error {
	if arg == "" {
		return fmt.Errorf("NODE is required")
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	res, err := selectnode.Resolve(cfg, arg, selectnode.Options{Buffer: opt.Buffer})
	if err != nil {
		return err
	}
	if len(res.Nodes) == 0 {
		return selectnode.NotFoundOne(res.SearchField, arg, res.ListName)
	}
	n := res.Nodes[0]
	if opt.Plain {
		fmt.Println(output.ShowPlain(n))
		return nil
	}
	fmt.Println(table.FromNodes([]*node.Node{n}, opt.Buffer))
	fmt.Println(n.Content)
	return nil
}
