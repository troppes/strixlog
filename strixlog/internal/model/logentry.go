package model

import (
	"fmt"
	"time"
)

// LogEntry is the normalised internal representation of a single log line.
type LogEntry struct {
	Timestamp time.Time
	Source    string // container name
	Line      string // raw log text
}

// String returns the normalised output format: [<timestamp>] [<source>] <line>
func (e LogEntry) String() string {
	return fmt.Sprintf("[%s] [%s] %s", e.Timestamp.UTC().Format(time.RFC3339), e.Source, e.Line)
}
