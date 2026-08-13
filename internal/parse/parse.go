// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package parse

import (
	"fmt"
	"os"
	"strings"

	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/labels"
	"github.com/sierra-tango-echo/alces-hunt/internal/node"
	"github.com/sierra-tango-echo/alces-hunt/internal/profile"
	"github.com/sierra-tango-echo/alces-hunt/internal/prompt"
	"github.com/sierra-tango-echo/alces-hunt/internal/store"
	"github.com/sierra-tango-echo/alces-hunt/internal/table"
)

// Options are CLI flags for parse.
type Options struct {
	Prefix        string
	Start         string
	Auto          bool
	AllowExisting bool
	SkipUsedIndex bool
	SkipSet       bool
	DryRun        bool
	DefaultLabel  string
}

// Run moves nodes from the buffer into the parsed list.
func Run(cfg *config.Config, opt Options) error {
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	buffer, err := store.Load(cfg.BufferDir())
	if err != nil {
		return err
	}
	if len(buffer.Nodes()) == 0 {
		return fmt.Errorf("No nodes in buffer")
	}
	parsed, err := store.Load(cfg.ParsedDir())
	if err != nil {
		return err
	}

	skipUsed := cfg.SkipUsedIndex
	if opt.SkipSet {
		skipUsed = opt.SkipUsedIndex
	}

	defaultLabel := opt.DefaultLabel
	if defaultLabel == "" {
		defaultLabel = cfg.DefaultLabel
	}
	switch strings.ToLower(defaultLabel) {
	case "long", "short", "blank":
		defaultLabel = strings.ToLower(defaultLabel)
	default:
		return fmt.Errorf("Invalid argument for 'default_label', must be 'long', 'short', or 'blank'")
	}

	used := append([]string{}, parsed.Labels()...)
	labelOpt := labels.Options{
		Prefix:       opt.Prefix,
		Start:        opt.Start,
		DefaultLabel: defaultLabel,
		DefaultStart: cfg.DefaultStart,
		PrefixStarts: cfg.PrefixStarts,
	}

	var final []*node.Node
	if opt.Auto {
		final, err = automatic(buffer.Nodes(), used, skipUsed, opt.AllowExisting || cfg.AllowExisting, parsed, labelOpt)
	} else {
		final, err = manual(buffer.Nodes(), used, labelOpt)
	}
	if err != nil {
		return err
	}
	if len(final) == 0 {
		return nil
	}

	if opt.DryRun {
		fmt.Println("Resulting node data (dry run):")
		fmt.Println(table.FromNodes(final, false))
		return nil
	}

	var existing []*node.Node
	for _, old := range parsed.Nodes() {
		for _, n := range final {
			if old.ID == n.ID {
				existing = append(existing, old)
			}
		}
	}
	if err := parsed.Delete(existing); err != nil {
		return err
	}
	apply := cfg.HasAutoApply()
	for _, n := range final {
		if err := buffer.Delete([]*node.Node{n}); err != nil {
			return err
		}
		n.AutoApply = apply
		parsed.Add(n)
	}
	if err := parsed.Save(); err != nil {
		return err
	}
	if err := buffer.Save(); err != nil {
		return err
	}
	fmt.Println("Nodes saved to parsed node list:")
	fmt.Println(table.FromNodes(final, false))

	if apply {
		for _, n := range final {
			if err := profile.ApplyRules(cfg, n.Label); err != nil {
				fmt.Printf("ERROR: %s\n", err)
			}
		}
	}
	return nil
}

func automatic(bufferNodes []*node.Node, used []string, skipUsed, allowExisting bool, parsed *store.List, opt labels.Options) ([]*node.Node, error) {
	presetCount := map[string]int{}
	for _, n := range bufferNodes {
		if p := n.PresetLabel(); p != "" {
			presetCount[p]++
		}
	}
	for _, c := range presetCount {
		if c > 1 {
			return nil, fmt.Errorf("Duplicate preset labels in buffer list. Please resolve any duplicates before continuing.")
		}
	}

	if !allowExisting {
		var existing []string
		for _, pn := range parsed.Nodes() {
			for _, bn := range bufferNodes {
				if pn.ID == bn.ID {
					existing = append(existing, pn.ID)
				}
			}
		}
		if len(existing) > 0 {
			return nil, fmt.Errorf("The following IDs already exist in the parsed list:\n%s", strings.Join(existing, "\n"))
		}
	}

	usedAuto := []string{}
	if skipUsed {
		usedAuto = append([]string{}, used...)
	}

	var newLabels []string
	for _, n := range bufferNodes {
		if label := n.PresetLabel(); label != "" {
			n.Label = label
			newLabels = append(newLabels, label)
			if skipUsed {
				usedAuto = append(usedAuto, label)
			}
		}
	}
	for _, n := range bufferNodes {
		if n.Label == "" {
			label := labels.AutoLabel(n, usedAuto, opt)
			n.Label = label
			usedAuto = append(usedAuto, label)
			newLabels = append(newLabels, label)
		}
	}

	all := append(append([]string{}, newLabels...), used...)
	if labels.Contains(all, "") {
		return nil, fmt.Errorf("One or more nodes generated a blank label, likely because '--default-label' was set to 'blank'.")
	}
	if dup := labels.FirstDuplicate(all); dup != "" {
		return nil, fmt.Errorf("The label %s was parsed for multiple nodes. Resolve duplicates or try using '--skip-used-index'", dup)
	}
	return bufferNodes, nil
}

func manual(bufferNodes []*node.Node, used []string, opt labels.Options) ([]*node.Node, error) {
	choices := make([]prompt.Choice, len(bufferNodes))
	for i, n := range bufferNodes {
		name := n.Hostname
		if n.IP != "" {
			if name != "" {
				name += " - "
			}
			name += n.IP
		}
		choices[i] = prompt.Choice{Name: name, Index: i}
	}

	var defaults []string
	usedSession := append([]string{}, used...)

	for {
		res, err := prompt.OrderedMultiSelect(os.Stdout, os.Stdin, "Select nodes:", choices, defaults)
		if err != nil {
			if err.Error() == "cancelled" {
				return nil, fmt.Errorf("Cancelled by user")
			}
			return nil, err
		}
		if res.Edit != nil {
			idx := res.Edit.Index
			n := bufferNodes[idx]
			reserved := append([]string{}, usedSession...)
			for _, c := range choices {
				if c.Label != "" {
					reserved = append(reserved, c.Label)
				}
			}
			prefill := n.PresetLabel()
			if prefill == "" {
				prefill = labels.AutoLabel(n, reserved, opt)
			}
			name, err := prompt.Ask(os.Stdout, os.Stdin,
				fmt.Sprintf("Enter the alias to be used as a label for node '%s'\nChoose label", n.Hostname),
				prefill)
			if err != nil {
				return nil, err
			}
			name = strings.TrimSpace(name)
			if name == "" {
				name = n.Hostname
			}
			if labels.Contains(reserved, name) && n.Label != name {
				return nil, fmt.Errorf("Label already exists")
			}
			n.Label = name
			choices[idx].Label = name
			usedSession = append(usedSession, name)
			defaults = append(defaults, choices[idx].Name)
			continue
		}

		var out []*node.Node
		for _, c := range res.Selected {
			n := bufferNodes[c.Index]
			if n.Label == "" {
				reserved := append([]string{}, usedSession...)
				prefill := n.PresetLabel()
				if prefill == "" {
					prefill = labels.AutoLabel(n, reserved, opt)
				}
				name, err := prompt.Ask(os.Stdout, os.Stdin,
					fmt.Sprintf("Enter the alias to be used as a label for node '%s'\nChoose label", n.Hostname),
					prefill)
				if err != nil {
					return nil, err
				}
				name = strings.TrimSpace(name)
				if name == "" {
					name = n.Hostname
				}
				n.Label = name
				usedSession = append(usedSession, name)
			}
			out = append(out, n)
		}
		return out, nil
	}
}
