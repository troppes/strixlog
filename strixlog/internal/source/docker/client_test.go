package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(ts *httptest.Server) *Client {
	return newClientWithBase(ts.URL+"/"+apiVersion, ts.Client().Transport)
}

func TestListContainers(t *testing.T) {
	containers := []Container{
		{ID: "abc123def456", Names: []string{"/web"}},
		{ID: "xyz789ghi012", Names: []string{"/db"}},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+apiVersion+"/containers/json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	got, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d containers, want 2", len(got))
	}
	if got[0].ID != "abc123def456" {
		t.Errorf("first container ID = %q, want abc123def456", got[0].ID)
	}
}

func TestListContainersServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.ListContainers(context.Background())
	if err == nil {
		t.Error("expected error on 500 response, got nil")
	}
}

func TestStreamLogs(t *testing.T) {
	payload := buildFrame(1, []byte("log line one\n"))
	payload = append(payload, buildFrame(1, []byte("log line two\n"))...)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/logs") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	body, err := client.StreamLogs(context.Background(), "testid")
	if err != nil {
		t.Fatalf("StreamLogs: %v", err)
	}
	defer body.Close()

	f1, err := ReadFrame(body)
	if err != nil {
		t.Fatalf("ReadFrame 1: %v", err)
	}
	if string(f1.Payload) != "log line one\n" {
		t.Errorf("frame 1 payload = %q, want %q", f1.Payload, "log line one\n")
	}

	f2, err := ReadFrame(body)
	if err != nil {
		t.Fatalf("ReadFrame 2: %v", err)
	}
	if string(f2.Payload) != "log line two\n" {
		t.Errorf("frame 2 payload = %q, want %q", f2.Payload, "log line two\n")
	}
}

func TestWatchEvents(t *testing.T) {
	events := []ContainerEvent{
		{Action: "start", Actor: struct {
			ID         string            `json:"ID"`
			Attributes map[string]string `json:"Attributes"`
		}{ID: "abc123", Attributes: map[string]string{"name": "web"}}},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/events") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		for _, e := range events {
			data, _ := json.Marshal(e)
			w.Write(append(data, '\n'))
		}
		// flush and let the handler return to close the stream
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer ts.Close()

	client := newTestClient(ts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := client.WatchEvents(ctx)
	if err != nil {
		t.Fatalf("WatchEvents: %v", err)
	}

	event, ok := <-ch
	if !ok {
		t.Fatal("channel closed before receiving event")
	}
	if event.Action != "start" {
		t.Errorf("event.Action = %q, want start", event.Action)
	}
	if event.Actor.ID != "abc123" {
		t.Errorf("event.Actor.ID = %q, want abc123", event.Actor.ID)
	}
}
