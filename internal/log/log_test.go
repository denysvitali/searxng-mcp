package log

import (
	"os"
	"testing"
)

func TestInit_LogsToStderr(t *testing.T) {
	Init("info")

	l := Get()
	if l.Out != os.Stderr {
		t.Fatalf("expected logger output to be os.Stderr, got %T", l.Out)
	}
	if l.Out == os.Stdout {
		t.Fatal("logger output must not be os.Stdout")
	}
}

