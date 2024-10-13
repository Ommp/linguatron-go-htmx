package main

import (
	"testing"
)

func TestGetNextEaseLevel(t *testing.T) {
	got := getNextEaseLevel(1, 2)
	want := 2

	if got != want {
		t.Errorf("got %d want %d", got, want)
	}
}
