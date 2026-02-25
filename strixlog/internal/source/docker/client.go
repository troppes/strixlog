package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
)

const (
	apiVersion = "v1.45"
	socketPath = "/var/run/docker.sock"
)

// Container is the subset of Docker's container summary we need.
type Container struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
}

// ContainerEvent represents a start or die event from the Docker events stream.
type ContainerEvent struct {
	Action string `json:"Action"` // "start" or "die"
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

// Client is a minimal Docker Engine API client over the Unix socket.
type Client struct {
	http *http.Client
	base string
}

// NewClient creates a Client that talks to the Docker daemon via the Unix socket.
func NewClient() *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		http: &http.Client{Transport: transport},
		base: "http://localhost/" + apiVersion,
	}
}

// newClientWithBase creates a Client with a custom base URL (for testing).
func newClientWithBase(base string, transport http.RoundTripper) *Client {
	return &Client{
		http: &http.Client{Transport: transport},
		base: base,
	}
}

// ListContainers returns all currently running containers.
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/containers/json", nil)
	if err != nil {
		return nil, fmt.Errorf("building list containers request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list containers: unexpected status %d", resp.StatusCode)
	}

	var containers []Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decoding containers: %w", err)
	}
	return containers, nil
}

// StreamLogs opens a log stream for the container with the given ID.
// The caller is responsible for closing the returned ReadCloser.
// The stream uses Docker's multiplexed frame format unless the container has a TTY.
func (c *Client) StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/containers/%s/logs?follow=true&stdout=true&stderr=true&tail=0", c.base, containerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building stream logs request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream logs: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("stream logs: unexpected status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// WatchEvents subscribes to Docker container start/die events.
// Returns a channel that receives events until ctx is cancelled or the stream ends.
func (c *Client) WatchEvents(ctx context.Context) (<-chan ContainerEvent, error) {
	filters, _ := json.Marshal(map[string][]string{"event": {"start", "die"}})
	params := url.Values{"filters": {string(filters)}}
	eventsURL := c.base + "/events?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building events request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("watch events: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("watch events: unexpected status %d", resp.StatusCode)
	}

	ch := make(chan ContainerEvent)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			var event ContainerEvent
			if err := json.Unmarshal(line, &event); err != nil {
				continue
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			log.Printf("events scanner error: %v", err)
		}
	}()

	return ch, nil
}
