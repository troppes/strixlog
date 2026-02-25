package printer

import (
	"context"
	"fmt"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

// PrintLogs reads log entries from ch and writes each to stdout until ctx is done.
func PrintLogs(ctx context.Context, ch <-chan model.LogEntry) {
	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println(entry.String())
		case <-ctx.Done():
			return
		}
	}
}
