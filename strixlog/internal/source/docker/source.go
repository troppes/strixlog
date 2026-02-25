package docker

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

// streamer holds the cancel func for one container's log stream goroutine.
type streamer struct {
	cancel context.CancelFunc
}

// DockerSource implements source.LogSource for Docker environments.
type DockerSource struct {
	cli      *dockerclient.Client
	hostname string
	logs     chan model.LogEntry

	mu       sync.Mutex
	streams  map[string]*streamer
	cancelFn context.CancelFunc
	stopOnce sync.Once
}

// NewDockerSource creates a DockerSource using the default Docker socket.
func NewDockerSource() (*DockerSource, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &DockerSource{
		cli:      cli,
		hostname: os.Getenv("HOSTNAME"),
		logs:     make(chan model.LogEntry, 256),
		streams:  make(map[string]*streamer),
	}, nil
}

// Start begins streaming logs from all running containers and watches for new ones.
// Start must not be called concurrently with Stop.
func (s *DockerSource) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.cancelFn = cancel
	s.mu.Unlock()

	containers, err := s.cli.ContainerList(ctx, container.ListOptions{})
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

	eventCh, errCh := s.cli.Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("event", "start"),
			filters.Arg("event", "die"),
		),
	})

	go s.handleEvents(ctx, eventCh, errCh)
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
		if s.cli != nil {
			s.cli.Close()
		}
	})
	return nil
}

// Logs returns the channel on which collected log entries are delivered.
func (s *DockerSource) Logs() <-chan model.LogEntry {
	return s.logs
}

func (s *DockerSource) handleEvents(ctx context.Context, eventCh <-chan events.Message, errCh <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errCh:
			if !ok {
				return
			}
			if ctx.Err() == nil {
				log.Printf("docker events error: %v", err)
			}
			return
		case event, ok := <-eventCh:
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
	if ctx.Err() != nil {
		return
	}
	body, err := s.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "0",
	})
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("container logs for %s: %v", name, err)
		}
		return
	}
	defer body.Close()

	pr, pw := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, body)
		pw.CloseWithError(err)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
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
			pr.CloseWithError(ctx.Err())
			return
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		log.Printf("log scanner for %s: %v", name, err)
	}
}
