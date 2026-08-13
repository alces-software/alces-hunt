// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Choice is one row in the ordered multi-select.
type Choice struct {
	Name  string
	Label string
	Index int
}

// MultiSelectResult is the outcome of an interactive selection.
type MultiSelectResult struct {
	// Selected is the chosen items in selection order.
	Selected []Choice
	// Edit is set when the user requested an early-return label edit.
	Edit *Choice
}

// OrderedMultiSelect presents a TTY multi-select.
// Space on an unselected item requests a label edit (early-return).
// Space on a selected item deselects it.
// Enter commits the current selection.
func OrderedMultiSelect(w io.Writer, r io.Reader, title string, choices []Choice, selectedNames []string) (*MultiSelectResult, error) {
	if !isTTY() {
		return lineMultiSelect(w, r, title, choices)
	}

	fd := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		return lineMultiSelect(w, r, title, choices)
	}
	defer term.Restore(fd, old)

	tw := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "")

	active := 0
	order := []int{}
	inOrder := map[int]bool{}
	for _, name := range selectedNames {
		for i, c := range choices {
			if c.Name == name && !inOrder[i] {
				order = append(order, i)
				inOrder[i] = true
			}
		}
	}

	redraw := func() {
		fmt.Fprintf(w, "\r\n%s\r\n", title)
		fmt.Fprintf(w, "(space: select/edit label, enter: confirm, q: abort)\r\n")
		for i, c := range choices {
			cursor := "  "
			if i == active {
				cursor = "‣ "
			}
			mark := "⬡"
			if inOrder[i] {
				mark = "⬢"
			}
			extra := ""
			if c.Label != "" {
				extra = " (" + c.Label + ")"
			}
			fmt.Fprintf(w, "%s%s %s%s\r\n", cursor, mark, c.Name, extra)
		}
	}

	for {
		redraw()
		b := make([]byte, 8)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return nil, err
		}
		key := b[:n]
		switch {
		case len(key) == 1 && (key[0] == 'q' || key[0] == 3): // q or Ctrl-C
			return nil, fmt.Errorf("cancelled")
		case len(key) == 1 && (key[0] == '\r' || key[0] == '\n'):
			var selected []Choice
			for _, i := range order {
				selected = append(selected, choices[i])
			}
			return &MultiSelectResult{Selected: selected}, nil
		case len(key) == 1 && key[0] == ' ':
			i := active
			if inOrder[i] {
				// deselect
				newOrder := order[:0]
				for _, j := range order {
					if j != i {
						newOrder = append(newOrder, j)
					}
				}
				order = newOrder
				inOrder[i] = false
				choices[i].Label = ""
				continue
			}
			c := choices[i]
			c.Index = i
			return &MultiSelectResult{Edit: &c}, nil
		case len(key) >= 3 && key[0] == 27 && key[1] == '[' && key[2] == 'A',
			len(key) == 1 && key[0] == 'k':
			if active > 0 {
				active--
			}
		case len(key) >= 3 && key[0] == 27 && key[1] == '[' && key[2] == 'B',
			len(key) == 1 && key[0] == 'j':
			if active < len(choices)-1 {
				active++
			}
		}
		_ = tw
	}
}

// Ask reads a line with a prefilled default. When stdin is a TTY the
// default is shown; an empty response keeps the default.
func Ask(w io.Writer, r io.Reader, question, value string) (string, error) {
	if value != "" {
		fmt.Fprintf(w, "%s [%s]: ", question, value)
	} else {
		fmt.Fprintf(w, "%s: ", question)
	}
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return value, nil
	}
	return line, nil
}

func lineMultiSelect(w io.Writer, r io.Reader, title string, choices []Choice) (*MultiSelectResult, error) {
	fmt.Fprintln(w, title)
	for i, c := range choices {
		fmt.Fprintf(w, "  [%d] %s\n", i+1, c.Name)
	}
	fmt.Fprint(w, "Enter selection (comma-separated numbers, or 'all'): ")
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return &MultiSelectResult{}, nil
	}
	var selected []Choice
	if strings.EqualFold(line, "all") {
		return &MultiSelectResult{Selected: append([]Choice{}, choices...)}, nil
	}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err != nil {
			continue
		}
		if n < 1 || n > len(choices) {
			continue
		}
		selected = append(selected, choices[n-1])
	}
	return &MultiSelectResult{Selected: selected}, nil
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
