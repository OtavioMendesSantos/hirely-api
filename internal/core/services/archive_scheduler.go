package services

import (
	"context"
	"hirely-api/internal/core/ports"
	"log/slog"
	"time"
	_ "time/tzdata"
)

// ArchiveClock makes daily scheduling and wake-up behavior deterministic in tests.
type ArchiveClock interface {
	Now() time.Time
	NewTimer(time.Duration) ArchiveTimer
}

type ArchiveTimer interface {
	C() <-chan time.Time
	Stop() bool
}

type systemArchiveClock struct{}

func (systemArchiveClock) Now() time.Time { return time.Now() }
func (systemArchiveClock) NewTimer(delay time.Duration) ArchiveTimer {
	return systemArchiveTimer{time.NewTimer(delay)}
}

type systemArchiveTimer struct{ *time.Timer }

func (timer systemArchiveTimer) C() <-chan time.Time { return timer.Timer.C }

type ArchiveScheduler struct {
	processor ports.ArchiveProcessor
	clock     ArchiveClock
	location  *time.Location
}

func NewArchiveScheduler(processor ports.ArchiveProcessor, clock ArchiveClock) (*ArchiveScheduler, error) {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return nil, err
	}
	if clock == nil {
		clock = systemArchiveClock{}
	}
	return &ArchiveScheduler{processor: processor, clock: clock, location: location}, nil
}

func NextArchiveRun(now time.Time, location *time.Location) time.Time {
	local := now.In(location)
	next := time.Date(local.Year(), local.Month(), local.Day(), 7, 0, 0, 0, location)
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// RunOnce is used synchronously on startup and by the daily loop. A timeout or
// failed batch is logged, without preventing the HTTP server from starting.
func (s *ArchiveScheduler) RunOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	now := s.clock.Now().UTC()
	total := 0
	slog.InfoContext(ctx, "Application auto-archive started", "operation", "AutoArchive")
	for {
		if err := ctx.Err(); err != nil {
			slog.ErrorContext(parent, "Application auto-archive stopped", "count", total, "error", err)
			return
		}
		count, err := s.processor.ArchiveDueBatch(ctx, now, 100)
		if err != nil {
			slog.ErrorContext(parent, "Application auto-archive failed", "count", total, "error", err)
			return
		}
		total += count
		if count < 100 {
			break
		}
	}
	slog.InfoContext(ctx, "Application auto-archive completed", "operation", "AutoArchive", "count", total)
}

// Run waits without polling. If suspended past 7am, the expired timer runs once
// on resumption and the next timer is calculated from the current wall clock.
func (s *ArchiveScheduler) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		now := s.clock.Now()
		next := NextArchiveRun(now, s.location)
		timer := s.clock.NewTimer(next.Sub(now))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
			timer.Stop()
			if ctx.Err() == nil {
				s.RunOnce(ctx)
			}
		}
	}
}
