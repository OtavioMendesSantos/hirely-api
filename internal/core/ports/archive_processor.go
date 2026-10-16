package ports

import (
	"context"
	"time"
)

// ArchiveProcessor atomically archives one bounded batch and its lifecycle events.
type ArchiveProcessor interface {
	ArchiveDueBatch(ctx context.Context, now time.Time, limit int) (int, error)
}
