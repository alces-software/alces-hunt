// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package output

import (
	"fmt"
	"strings"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

// ListPlain is the --plain format for list: id hostname ip groups label presets.
func ListPlain(n *node.Node) string {
	return strings.Join([]string{
		n.ID,
		n.Hostname,
		n.IP,
		n.PlainGroups(),
		n.Label,
		n.CompactPresetsJSON(),
	}, "\t")
}

// ShowPlain is the --plain format for show: id hostname ip groups content.
func ShowPlain(n *node.Node) string {
	return strings.Join([]string{
		n.ID,
		n.Hostname,
		n.IP,
		n.PlainGroups(),
		n.Content,
	}, "\t")
}

// ListPlainAll emits one record per node, already sorted by the caller.
func ListPlainAll(nodes []*node.Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ListPlain(n))
	}
	if len(nodes) > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

// FormatError prefixes a message for stderr.
func FormatError(err error) string {
	return fmt.Sprintf("%s\n", err.Error())
}
