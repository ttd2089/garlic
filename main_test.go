package main

import (
	"testing"
)

func TestSanity(t *testing.T) {
	if 2+2 != 4 {
		t.Fatalf("Winston, no!")
	}
}
