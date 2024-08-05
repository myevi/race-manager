package main

import (
	"testing"
)

func TestMain(t *testing.T) {
	res, err := parsePDF("silverstone2024.pdf")
	if err != nil {
		t.Log("some err: %w", err)
		t.Fail()
	}

	if res == nil {
		t.Log("empty result")
		t.Fail()
	}
}