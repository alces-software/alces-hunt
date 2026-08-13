// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// WriteIfConfigured writes the current PID when ALCES_HUNT_pidfile is set.
// Returns a cleanup function.
func WriteIfConfigured() (func(), error) {
	path := os.Getenv("ALCES_HUNT_pidfile")
	if path == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing pidfile %s: %w", path, err)
	}
	return func() { os.Remove(path) }, nil
}
