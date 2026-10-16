package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type testArchiveTimer struct {
	channel chan time.Time
	stopped chan struct{}
	once    sync.Once
}

func (timer *testArchiveTimer) C() <-chan time.Time { return timer.channel }
func (timer *testArchiveTimer) Stop() bool {
	timer.once.Do(func() { close(timer.stopped) })
	return true
}

type testArchiveClock struct {
	mu      sync.Mutex
	now     time.Time
	created chan *testArchiveTimer
	delays  chan time.Duration
}

func (clock *testArchiveClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}
func (clock *testArchiveClock) set(now time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = now
}
func (clock *testArchiveClock) NewTimer(delay time.Duration) ArchiveTimer {
	timer := &testArchiveTimer{channel: make(chan time.Time, 1), stopped: make(chan struct{})}
	clock.delays <- delay
	clock.created <- timer
	return timer
}

type testArchiveProcessor struct {
	counts []int
	calls  int
	err    error
	called chan time.Time
	cancel context.CancelFunc
}

func (processor *testArchiveProcessor) ArchiveDueBatch(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit != 100 {
		panic("wrong batch size")
	}
	processor.calls++
	if processor.called != nil {
		processor.called <- now
	}
	if processor.cancel != nil {
		processor.cancel()
	}
	if processor.err != nil {
		return 0, processor.err
	}
	if len(processor.counts) == 0 {
		return 0, nil
	}
	count := processor.counts[0]
	processor.counts = processor.counts[1:]
	return count, nil
}

func TestNextArchiveRunUsesSaoPauloAtSeven(t *testing.T) {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ now, want string }{
		{"2026-10-16T09:59:59Z", "2026-10-16T10:00:00Z"},
		{"2026-10-16T10:00:00Z", "2026-10-17T10:00:00Z"},
		{"2026-12-31T23:00:00Z", "2027-01-01T10:00:00Z"},
		// Historical daylight saving verifies calendar scheduling rather than fixed 24h waits.
		{"2018-11-03T10:00:00Z", "2018-11-04T09:00:00Z"},
	}
	for _, test := range tests {
		now, _ := time.Parse(time.RFC3339, test.now)
		got := NextArchiveRun(now, location).UTC().Format(time.RFC3339)
		if got != test.want {
			t.Fatalf("%s: want %s got %s", test.now, test.want, got)
		}
	}
}

func TestStartupArchiveBatchesAndFailure(t *testing.T) {
	for _, test := range []struct {
		name   string
		counts []int
		err    error
		want   int
	}{
		{"bounded batches", []int{100, 100, 3}, nil, 3},
		{"failure", nil, errors.New("database asleep"), 1},
		{"nothing due", nil, nil, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			processor := &testArchiveProcessor{counts: test.counts, err: test.err}
			scheduler, err := NewArchiveScheduler(processor, nil)
			if err != nil {
				t.Fatal(err)
			}
			scheduler.RunOnce(context.Background())
			if processor.calls != test.want {
				t.Fatalf("got %d calls", processor.calls)
			}
		})
	}
	processor := &testArchiveProcessor{counts: []int{100}}
	ctx, cancel := context.WithCancel(context.Background())
	processor.cancel = cancel
	scheduler, _ := NewArchiveScheduler(processor, nil)
	scheduler.RunOnce(ctx)
	if processor.calls != 1 {
		t.Fatal("shutdown did not stop the next batch")
	}
}

func TestDailyArchiveResumesAfterSleepWithoutPolling(t *testing.T) {
	before, _ := time.Parse(time.RFC3339, "2026-10-16T09:00:00Z")
	resumed, _ := time.Parse(time.RFC3339, "2026-10-19T15:00:00Z")
	clock := &testArchiveClock{now: before, created: make(chan *testArchiveTimer, 2), delays: make(chan time.Duration, 2)}
	processor := &testArchiveProcessor{called: make(chan time.Time, 1)}
	scheduler, _ := NewArchiveScheduler(processor, clock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); scheduler.Run(ctx) }()
	timer := <-clock.created
	if delay := <-clock.delays; delay != time.Hour {
		t.Fatalf("wrong initial wait %v", delay)
	}
	select {
	case <-processor.called:
		t.Fatal("processed before the scheduled time")
	default:
	}
	clock.set(resumed)
	timer.channel <- resumed
	if processed := <-processor.called; !processed.Equal(resumed) {
		t.Fatalf("processed stale date: %v", processed)
	}
	nextTimer := <-clock.created
	if delay := <-clock.delays; delay != 19*time.Hour {
		t.Fatalf("wrong resumption wait %v", delay)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}
	select {
	case <-nextTimer.stopped:
	default:
		t.Fatal("timer leaked on shutdown")
	}
}
