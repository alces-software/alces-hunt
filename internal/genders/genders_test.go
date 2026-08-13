// Copyright (C) 2026 Alces Software Ltd.
// SPDX-License-Identifier: EPL-2.0

package genders

import (
	"reflect"
	"testing"
)

func TestExpandSimple(t *testing.T) {
	got, err := Expand("node[1-3]")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"node1", "node2", "node3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestExpandMixedPadded(t *testing.T) {
	got, err := Expand("c[01-02],login[1-1]")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c01", "c02", "login1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestExpandInvalidRange(t *testing.T) {
	_, err := Expand("node[5-1]")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*InvalidRangeError); !ok {
		t.Fatalf("want InvalidRangeError, got %T %v", err, err)
	}
	if err.Error() == "" || err.Error()[:len("InvalidRangeError")] != "InvalidRangeError" {
		t.Fatalf("message should name InvalidRangeError: %v", err)
	}
}

func TestExpandInvalidFormat(t *testing.T) {
	_, err := Expand("node[abc]")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExpandTrimSpaces(t *testing.T) {
	got, err := Expand("node1, node2")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"node1", "node2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSplitRegexNoExpansion(t *testing.T) {
	got := SplitRegex("node[1-3],login.*")
	want := []string{"node[1-3]", "login.*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
