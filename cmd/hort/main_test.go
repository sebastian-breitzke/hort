package main

import (
	"strings"
	"testing"
)

func TestWriteValueFromStdinPreservesExactValue(t *testing.T) {
	got, err := writeValue("", true, strings.NewReader("secret-without-argv"))
	if err != nil {
		t.Fatalf("writeValue: %v", err)
	}
	if got != "secret-without-argv" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteValueRejectsAmbiguousAndEmptyInput(t *testing.T) {
	for _, test := range []struct {
		name       string
		flagValue  string
		valueStdin bool
		stdin      string
	}{
		{name: "both", flagValue: "argument", valueStdin: true, stdin: "stdin"},
		{name: "neither"},
		{name: "empty stdin", valueStdin: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := writeValue(test.flagValue, test.valueStdin, strings.NewReader(test.stdin)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
