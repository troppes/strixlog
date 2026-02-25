package source

import (
	"context"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

// LogSource is the abstraction all runtime implementations must satisfy.
type LogSource interface {
	Start(ctx context.Context) error
	Stop() error
	Logs() <-chan model.LogEntry
}
