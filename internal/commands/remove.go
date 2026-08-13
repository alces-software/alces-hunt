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
	"github.com/sierra-tango-echo/alces-hunt/internal/selectnode"
	"github.com/sierra-tango-echo/alces-hunt/internal/table"
)

// SelectOptions are shared flags for node-selecting commands.
type SelectOptions struct {
	Buffer        bool
	Regex         bool
	MatchHostname bool
}

// RemoveNode deletes matching nodes from a list.
func RemoveNode(cfg *config.Config, arg string, opt SelectOptions) error {
	if arg == "" {
		return fmt.Errorf("NODE is required")
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	res, err := selectnode.Resolve(cfg, arg, selectnode.Options{
		Buffer:        opt.Buffer,
		Regex:         opt.Regex,
		MatchHostname: opt.MatchHostname,
	})
	if err != nil {
		return err
	}
	if len(res.Nodes) == 0 {
		return selectnode.NotFoundCollection(res.SearchField, res.ListName, arg)
	}
	if err := res.List.Delete(res.Nodes); err != nil {
		return err
	}
	if err := res.List.Save(); err != nil {
		return err
	}
	fmt.Printf("The following nodes have successfully been removed from list '%s'\n", res.ListName)
	fmt.Println(table.FromNodes(res.Nodes, opt.Buffer))
	return nil
}
