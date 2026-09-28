//go:build darwin

package main

import (
	"reflect"
	"testing"
)

func TestOrderedArgsAcceptsOptionsAfterIP(t *testing.T) {
	got := orderedArgs([]string{"192.0.2.10", "--port", "2200", "--user=ops"})
	want := []string{"--port", "2200", "--user=ops", "192.0.2.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderedArgs = %q, want %q", got, want)
	}
}
