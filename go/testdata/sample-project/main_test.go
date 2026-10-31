package main

import "testing"

func TestHello(t *testing.T) {
	expected := "Hello, World!"
	if got := Hello(); got != expected {
		t.Errorf("Hello() = %q, want %q", got, expected)
	}
}
