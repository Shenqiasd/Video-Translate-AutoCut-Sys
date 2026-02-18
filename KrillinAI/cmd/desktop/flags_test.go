package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() failed: %v", err)
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("io.Copy() failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() failed: %v", err)
	}

	return buffer.String()
}

func TestPrintDiagnoseShowsEffectiveLogDir(t *testing.T) {
	output := captureStdout(t, printDiagnose)
	if !strings.Contains(output, "path.effective_log_dir:") {
		t.Fatalf("printDiagnose() output missing effective log dir: %s", output)
	}
}
