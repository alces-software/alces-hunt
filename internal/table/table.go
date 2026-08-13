// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package table

import (
	"strings"
	"unicode/utf8"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

// FromNodes builds a Unicode box-drawing table for the given nodes.
func FromNodes(nodes []*node.Node, buffer bool) string {
	var headers []string
	if buffer {
		headers = []string{"ID", "Hostname", "IP", "Groups", "Presets"}
	} else {
		headers = []string{"ID", "Hostname", "IP", "Groups", "Label"}
	}
	rows := make([][]string, 0, len(nodes))
	for _, n := range nodes {
		if buffer {
			rows = append(rows, []string{n.ID, n.Hostname, n.IP, n.PrettyGroups(), n.PrettyPresets()})
		} else {
			rows = append(rows, []string{n.ID, n.Hostname, n.IP, n.PrettyGroups(), n.Label})
		}
	}
	return Render(headers, rows)
}

// Render draws a Unicode table with 1-column padding and multiline cells.
func Render(headers []string, rows [][]string) string {
	cols := len(headers)
	widths := make([]int, cols)
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}

	// split[row][col][line]
	split := make([][][]string, len(rows))
	for r, row := range rows {
		split[r] = make([][]string, cols)
		for c := 0; c < cols; c++ {
			val := ""
			if c < len(row) {
				val = row[c]
			}
			lines := strings.Split(val, "\n")
			split[r][c] = lines
			for _, line := range lines {
				if w := utf8.RuneCountInString(line); w > widths[c] {
					widths[c] = w
				}
			}
		}
	}

	var b strings.Builder
	writeSep(&b, widths, "┌", "┬", "┐")
	writeRow(&b, headers, widths)
	writeSep(&b, widths, "├", "┼", "┤")
	for _, row := range split {
		height := 1
		for _, lines := range row {
			if len(lines) > height {
				height = len(lines)
			}
		}
		for i := 0; i < height; i++ {
			vals := make([]string, cols)
			for c := 0; c < cols; c++ {
				if i < len(row[c]) {
					vals[c] = row[c][i]
				}
			}
			writeRow(&b, vals, widths)
		}
	}
	writeSep(&b, widths, "└", "┴", "┘")
	return strings.TrimRight(b.String(), "\n")
}

func writeSep(b *strings.Builder, widths []int, left, mid, right string) {
	b.WriteString(left)
	for i, w := range widths {
		if i > 0 {
			b.WriteString(mid)
		}
		b.WriteString(strings.Repeat("─", w+2))
	}
	b.WriteString(right)
	b.WriteByte('\n')
}

func writeRow(b *strings.Builder, vals []string, widths []int) {
	b.WriteString("│")
	for i, w := range widths {
		val := ""
		if i < len(vals) {
			val = vals[i]
		}
		b.WriteByte(' ')
		b.WriteString(val)
		pad := w - utf8.RuneCountInString(val)
		if pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}
		b.WriteByte(' ')
		b.WriteString("│")
	}
	b.WriteByte('\n')
}
