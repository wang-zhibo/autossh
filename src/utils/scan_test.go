package utils

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func withInputReader(t *testing.T, input string) {
	t.Helper()
	previous := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() {
		stdinReader = previous
	})
}

func TestScanlnReadsBufferedLinesWithoutLoss(t *testing.T) {
	withInputReader(t, "first line\nsecond line\n")

	var first, second string
	if err := Scanln(&first); err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if err := Scanln(&second); err != nil {
		t.Fatalf("read second line: %v", err)
	}

	if first != "first line" || second != "second line" {
		t.Fatalf("got %q and %q", first, second)
	}
}

func TestScanlnReturnsEOF(t *testing.T) {
	withInputReader(t, "")

	var input string
	if err := Scanln(&input); err != io.EOF {
		t.Fatalf("Scanln() error = %v, want io.EOF", err)
	}
}

func TestScanlnAcceptsFinalLineWithoutNewline(t *testing.T) {
	withInputReader(t, "exit")

	var input string
	if err := Scanln(&input); err != nil {
		t.Fatalf("read final line: %v", err)
	}
	if input != "exit" {
		t.Fatalf("input = %q, want exit", input)
	}
}
