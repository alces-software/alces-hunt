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

// ModifyLabel changes a parsed node's label.
func ModifyLabel(cfg *config.Config, oldLabel, newLabel string) error {
	if oldLabel == "" || newLabel == "" {
		return fmt.Errorf("OLD_LABEL and NEW_LABEL are required")
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	list, err := store.Load(cfg.ParsedDir())
	if err != nil {
		return err
	}
	if list.FindByLabel(newLabel) != nil {
		return fmt.Errorf("Label '%s' already exists in list '%s'", newLabel, list.Name())
	}
	n := list.FindByLabel(oldLabel)
	if n == nil {
		return fmt.Errorf("Node '%s' does not exist in list '%s'", oldLabel, list.Name())
	}
	n.Label = newLabel
	if err := list.Save(); err != nil {
		return err
	}
	fmt.Printf("Node '%s' in list '%s' relabeled to '%s'\n", oldLabel, list.Name(), newLabel)
	return nil
}
