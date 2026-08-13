// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
)

// ApplyRules finds the first auto_apply regex matching label and runs
// `<profile_command> apply <label> <identity>`. Failures are returned
// so the caller can log them; they must not abort parse.
func ApplyRules(cfg *config.Config, label string) error {
	if !cfg.HasAutoApply() {
		return nil
	}
	var identity string
	var rule string
	for _, r := range cfg.AutoApply {
		compiled, err := regexp.Compile(config.NormalizeRegex(r.Regex))
		if err != nil {
			continue
		}
		if compiled.MatchString(label) {
			identity = r.Identity
			rule = r.Regex
			break
		}
	}
	if identity == "" {
		return nil
	}
	fmt.Printf("Node %s matches auto-apply rule '%s: %s'\n", label, rule, identity)
	return Apply(cfg, label, identity)
}

// Apply executes the profile command. An unusable command is an error.
func Apply(cfg *config.Config, label, identity string) error {
	cmd := cfg.ProfileCommand
	if len(cmd) == 0 {
		return fmt.Errorf("Profile command is not defined")
	}
	path := cmd[0]
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("Could not find '%s'", path)
	}
	if st.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", path)
	}
	args := append(append([]string{}, cmd[1:]...), "apply", label, identity)
	c := exec.Command(path, args...)
	c.Env = os.Environ()
	logDir := filepath.Join(cfg.Root, "var", "log")
	_ = os.MkdirAll(logDir, 0o755)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("ERROR: %s: %s", err, string(out))
	}
	return nil
}
