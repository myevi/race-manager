package main

import (
	"testing"
)

func TestMain(t *testing.T) {
	err := parsePDF("sources/silverstone2024.pdf")
	if err != nil {
		t.Log("some err", err.Error())
		t.Fail()
	}
}
