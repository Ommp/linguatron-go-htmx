package main

import "testing"

func TestHello(t *testing.T) {
	got := startMessage()
	want := "Starting app..."

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
