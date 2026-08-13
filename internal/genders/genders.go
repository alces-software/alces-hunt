// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package genders

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// InvalidRangeError is raised for malformed or inverted bracket ranges.
type InvalidRangeError struct {
	Message string
}

func (e *InvalidRangeError) Error() string {
	return "InvalidRangeError: " + e.Message
}

var rangeRE = regexp.MustCompile(`^\[[0-9]+-[0-9]+\]$`)

// Expand splits a comma-separated genders-style list and expands brackets.
// Surrounding whitespace around each term is trimmed.
func Expand(str string) ([]string, error) {
	if strings.TrimSpace(str) == "" {
		return nil, nil
	}
	parts := strings.Split(str, ",")
	var out []string
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		expanded, err := expandBrackets(part)
		if err != nil {
			return nil, err
		}
		for _, item := range expanded {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

// SplitRegex splits a comma-separated regex list without expanding brackets.
func SplitRegex(str string) []string {
	parts := strings.Split(str, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func expandBrackets(str string) ([]string, error) {
	start := strings.IndexByte(str, '[')
	end := strings.IndexByte(str, ']')
	if start < 0 || end < 0 || end < start {
		return []string{str}, nil
	}
	contents := str[start : end+1]
	left := str[:start]
	right := str[end+1:]

	if !rangeRE.MatchString(contents) {
		return nil, &InvalidRangeError{Message: fmt.Sprintf("'%s' is not of the format [START-END].", contents)}
	}
	inner := contents[1 : len(contents)-1]
	nums := strings.SplitN(inner, "-", 2)
	first, err1 := strconv.Atoi(nums[0])
	last, err2 := strconv.Atoi(nums[1])
	if err1 != nil || err2 != nil {
		return nil, &InvalidRangeError{Message: fmt.Sprintf("'%s' is not of the format [START-END].", contents)}
	}
	if first > last {
		return nil, &InvalidRangeError{Message: fmt.Sprintf("'%s' has a start index that is greater than its end index.", contents)}
	}
	width := len(nums[0])
	if len(nums[1]) > width {
		width = len(nums[1])
	}
	// Preserve padding only when the start token is zero-padded / fixed-width.
	pad := strings.HasPrefix(nums[0], "0") && len(nums[0]) > 1 || len(nums[0]) == len(nums[1]) && len(nums[0]) > 1
	if !pad {
		width = 0
	}
	var out []string
	for i := first; i <= last; i++ {
		var idx string
		if width > 0 {
			idx = fmt.Sprintf("%0*d", width, i)
		} else {
			idx = strconv.Itoa(i)
		}
		out = append(out, left+idx+right)
	}
	return out, nil
}
