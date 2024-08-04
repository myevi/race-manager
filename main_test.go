package main

import "testing"

func TestMain(t *testing.T) {
	_, err := parsePDF("silverstone2024.pdf")
	t.Errorf("some err: %w", err)
}