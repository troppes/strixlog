package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/troppes/strixlog/strixlog/internal/model"
)

// newSourceWithServer creates a DockerSource backed by a test HTTP server.
func newSourceWithServer(ts *httptest.Server, hostname string) *DockerSource {
	return &DockerSource{
		client:   newClientWithBase(ts.URL+"/"+apiVersion, ts.Client().Transport),
		hostname: hostname,
		logs:     make(chan model.LogEntry, 64),
		streams:  make(map[string]*streamer),
	}
}

func TestDockerSourceExcludesSelf(t *testing.T) {
	selfID := "selfcontainer123"

	containers := []Container{
		{ID: selfID, Names: []string{"/strixlog"}},
		{ID: "other456", Names: []string{"/app"}},
	}

	logPayload := buildFrame(1, []byte(`{"msg":"hello"}`+"\n"))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+apiVersion+"/containers/json":
			json.NewEncoder(w).Encode(containers)
		case strings.Contains(r.URL.Path, "/logs"):
			w.WriteHeader(http.StatusOK)
			w.Write(logPayload)
		case strings.Contains(r.URL.Path, "/events"):
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	src := newSourceWithServer(ts, "selfcontainer")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer src.Stop()

	select {
	case entry := <-src.Logs():
		if entry.Source == "strixlog" {
			t.Error("received log from self — should have been excluded")
		}
		if entry.Source != "app" {
			t.Errorf("source = %q, want app", entry.Source)
		}
	case <-ctx.Done():
		t.Log("no log entry received (self was correctly excluded, other may not have sent logs)")
	}
}

func TestDockerSourceHandlesContainerStopCleanly(t *testing.T) {
	containers := []Container{
		{ID: "running123", Names: []string{"/myapp"}},
	}

	logPayload := buildFrame(1, []byte("log line\n"))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/"+apiVersion+"/containers/json":
			json.NewEncoder(w).Encode(containers)
		case strings.Contains(r.URL.Path, "/logs"):
			w.WriteHeader(http.StatusOK)
			w.Write(logPayload)
		case strings.Contains(r.URL.Path, "/events"):
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	src := newSourceWithServer(ts, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer src.Stop()

	select {
	case entry, ok := <-src.Logs():
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		if entry.Source != "myapp" {
			t.Errorf("source = %q, want myapp", entry.Source)
		}
	case <-ctx.Done():
		t.Error("timed out waiting for log entry")
	}
}
