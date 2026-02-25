package model

import (
	"testing"
	"time"
)

func TestLogEntryString(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		entry LogEntry
		want  string
	}{
		{
			name:  "basic entry",
			entry: LogEntry{Timestamp: ts, Source: "mycontainer", Line: "hello world"},
			want:  "[2024-01-15T10:30:00Z] [mycontainer] hello world",
		},
		{
			name:  "json log line",
			entry: LogEntry{Timestamp: ts, Source: "randomlog", Line: `{"level":"info","message":"started"}`},
			want:  `[2024-01-15T10:30:00Z] [randomlog] {"level":"info","message":"started"}`,
		},
		{
			name:  "empty line",
			entry: LogEntry{Timestamp: ts, Source: "app", Line: ""},
			want:  "[2024-01-15T10:30:00Z] [app] ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.entry.String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
