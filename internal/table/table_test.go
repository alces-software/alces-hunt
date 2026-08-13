// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package table

import (
	"strings"
	"testing"

	"github.com/sierra-tango-echo/alces-hunt/internal/node"
)

func TestFromNodesHeaders(t *testing.T) {
	n := node.New("id1", "host", "1.2.3.4", "", []string{"g"}, map[string]string{"label": "x"})
	buf := FromNodes([]*node.Node{n}, true)
	if !strings.Contains(buf, "Presets") || !strings.Contains(buf, "label: 'x'") {
		t.Fatalf("buffer table:\n%s", buf)
	}
	if !strings.Contains(buf, "┌") || !strings.Contains(buf, "│") {
		t.Fatalf("expected unicode box drawing:\n%s", buf)
	}
	parsed := FromNodes([]*node.Node{n}, false)
	if !strings.Contains(parsed, "Label") || strings.Contains(parsed, "Presets") {
		t.Fatalf("parsed table:\n%s", parsed)
	}
}
