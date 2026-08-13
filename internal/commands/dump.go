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
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
)

// DumpBuffer empties the buffer list.
func DumpBuffer(cfg *config.Config) error {
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	list, err := store.Load(cfg.BufferDir())
	if err != nil {
		return err
	}
	if err := list.Empty(); err != nil {
		return err
	}
	fmt.Println("Node buffer emptied.")
	return nil
}
