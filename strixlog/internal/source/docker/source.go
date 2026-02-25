package docker

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

// streamer holds the cancel func for one container's log stream goroutine.
// A pointer is stored in the streams map so goroutines can compare identity.
type streamer struct {
	cancel context.CancelFunc
}

// DockerSource implements source.LogSource for Docker environments.
type DockerSource struct {
	client   *Client
	hostname string
	logs     chan model.LogEntry

	mu       sync.Mutex
	streams  map[string]*streamer // containerID -> streamer
	cancelFn context.CancelFunc
	stopOnce sync.Once
}

// NewDockerSource creates a DockerSource using the default Docker socket.
func NewDockerSource() *DockerSource {
	return &DockerSource{
		client:   NewClient(),
		hostname: os.Getenv("HOSTNAME"),
		logs:     make(chan model.LogEntry, 256),
		streams:  make(map[string]*streamer),
	}
}

// Start begins streaming logs from all running containers and watches for new ones.
// Start must not be called concurrently with Stop.
func (s *DockerSource) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.cancelFn = cancel
	s.mu.Unlock()

	containers, err := s.client.ListContainers(ctx)
	if err != nil {
		cancel()
		return err
	}

	for _, c := range containers {
		if isSelf(c.ID, s.hostname) {
			continue
		}
		s.startStreamer(ctx, c.ID, containerName(c))
	}

	events, err := s.client.WatchEvents(ctx)
	if err != nil {
		cancel()
		return err
	}

	go s.handleEvents(ctx, events)
	return nil
}

// Stop cancels all active streams. Safe to call multiple times.
func (s *DockerSource) Stop() error {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		fn := s.cancelFn
		s.mu.Unlock()
		if fn != nil {
			fn()
		}
	})
	return nil
}

// Logs returns the channel on which collected log entries are delivered.
func (s *DockerSource) Logs() <-chan model.LogEntry {
	return s.logs
}

func (s *DockerSource) handleEvents(ctx context.Context, events <-chan ContainerEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			switch event.Action {
			case "start":
				id := event.Actor.ID
				if isSelf(id, s.hostname) {
					break
				}
				name := event.Actor.Attributes["name"]
				s.startStreamer(ctx, id, name)
			case "die":
				s.stopStreamer(event.Actor.ID)
			}
		}
	}
}

func (s *DockerSource) startStreamer(ctx context.Context, id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[id]; exists {
		return
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sr := &streamer{cancel: cancel}
	s.streams[id] = sr

	go func() {
		defer func() {
			s.mu.Lock()
			// Only remove our own entry — a container restart may have already
			// registered a new streamer for the same ID.
			if s.streams[id] == sr {
				delete(s.streams, id)
			}
			s.mu.Unlock()
		}()
		s.streamContainer(streamCtx, id, name)
	}()
}

func (s *DockerSource) stopStreamer(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sr, exists := s.streams[id]; exists {
		sr.cancel()
		delete(s.streams, id)
	}
}

func (s *DockerSource) streamContainer(ctx context.Context, id, name string) {
	body, err := s.client.StreamLogs(ctx, id)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("stream logs for %s: %v", name, err)
		}
		return
	}
	defer body.Close()

	for {
		frame, err := ReadFrame(body)
		if err != nil {
			if errors.Is(err, io.EOF) || isClosedError(err) || ctx.Err() != nil {
				return
			}
			log.Printf("read frame for %s: %v", name, err)
			return
		}

		line := strings.TrimRight(string(frame.Payload), "\n\r")
		if line == "" {
			continue
		}

		entry := model.LogEntry{
			Timestamp: time.Now().UTC(),
			Source:    name,
			Line:      line,
		}

		select {
		case s.logs <- entry:
		case <-ctx.Done():
			return
		}
	}
}

// isClosedError detects errors that occur when a connection is closed,
// typically during context cancellation or container stop.
func isClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe)
}
