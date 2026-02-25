package printer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

func TestPrintLogs(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entries := []model.LogEntry{
		{Timestamp: ts, Source: "web", Line: "request received"},
		{Timestamp: ts, Source: "db", Line: "query executed"},
	}

	ch := make(chan model.LogEntry, len(entries))
	for _, e := range entries {
		ch <- e
	}
	close(ch)

	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	PrintLogs(ctx, ch)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	for _, e := range entries {
		expected := fmt.Sprintf("[2024-01-15T10:30:00Z] [%s] %s", e.Source, e.Line)
		if !strings.Contains(output, expected) {
			t.Errorf("output missing line %q\ngot:\n%s", expected, output)
		}
	}
}

func TestPrintLogsStopsOnContextCancel(t *testing.T) {
	ch := make(chan model.LogEntry) // unbuffered, no sender

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		PrintLogs(ctx, ch)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Error("PrintLogs did not stop after context cancellation")
	}
}
