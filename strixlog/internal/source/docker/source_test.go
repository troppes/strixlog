package docker

import (
	"context"
	"testing"
	"time"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

func TestDockerSourceStopIsIdempotent(t *testing.T) {
	src := &DockerSource{
		logs:    make(chan model.LogEntry, 8),
		streams: make(map[string]*streamer),
	}

	_, cancel := context.WithCancel(context.Background())
	src.mu.Lock()
	src.cancelFn = cancel
	src.mu.Unlock()

	// Calling Stop multiple times must not panic.
	if err := src.Stop(); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := src.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
}

func TestStartStreamerDeduplication(t *testing.T) {
	src := &DockerSource{
		logs:    make(chan model.LogEntry, 8),
		streams: make(map[string]*streamer),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// startStreamer for a container that never sends data — just verify
	// that calling it twice for the same ID doesn't register two goroutines.
	//
	// We inject a context that is already cancelled so the streamer exits
	// immediately without needing a real Docker daemon.
	alreadyCancelled, c := context.WithCancel(ctx)
	c()

	src.startStreamer(alreadyCancelled, "abc123", "myapp")
	src.startStreamer(alreadyCancelled, "abc123", "myapp") // duplicate — must be a no-op

	// Give goroutines a moment to exit.
	time.Sleep(50 * time.Millisecond)

	src.mu.Lock()
	n := len(src.streams)
	src.mu.Unlock()

	// After exit the entry is cleaned up; either 0 or 1 is acceptable
	// (goroutine may have already cleaned up), but never 2.
	if n > 1 {
		t.Errorf("streams map has %d entries after dedup; want ≤1", n)
	}
}

func TestStopStreamerCleansUp(t *testing.T) {
	src := &DockerSource{
		logs:    make(chan model.LogEntry, 8),
		streams: make(map[string]*streamer),
	}

	called := false
	sr := &streamer{cancel: func() { called = true }}
	src.mu.Lock()
	src.streams["abc123"] = sr
	src.mu.Unlock()

	src.stopStreamer("abc123")

	if !called {
		t.Error("cancel was not called by stopStreamer")
	}

	src.mu.Lock()
	_, still := src.streams["abc123"]
	src.mu.Unlock()

	if still {
		t.Error("stream entry was not removed from map")
	}
}
