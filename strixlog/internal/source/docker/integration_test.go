//go:build integration

package docker

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"
)

// TestIntegrationDockerSource verifies that DockerSource streams logs
// from the real randomlog container.
//
// Requires Docker and docker compose to be running.
// Run with: go test -tags=integration ./internal/source/docker/...
func TestIntegrationDockerSource(t *testing.T) {
	t.Log("starting randomlog container via docker compose")

	// Bring up only randomlog (strixlog talks to Docker directly in this test)
	cmd := exec.Command("docker", "compose", "-f", "../../../../docker-compose.yml", "up", "-d", "randomlog")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker compose up: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		exec.Command("docker", "compose", "-f", "../../../../docker-compose.yml", "down").Run()
	})

	// Give randomlog a moment to start
	time.Sleep(2 * time.Second)

	src := NewDockerSource()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := src.Start(ctx); err != nil {
		t.Fatalf("DockerSource.Start: %v", err)
	}
	defer src.Stop()

	t.Log("waiting for a log entry from randomlog...")

	for {
		select {
		case entry, ok := <-src.Logs():
			if !ok {
				t.Fatal("log channel closed unexpectedly")
			}
			t.Logf("received: %s", entry)

			if entry.Source == "" {
				t.Error("entry.Source is empty")
			}

			// Verify the line is valid JSON matching randomlog's schema
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(entry.Line), &payload); err != nil {
				t.Logf("non-JSON line (may be startup noise): %s", entry.Line)
				continue
			}

			for _, field := range []string{"timestamp", "level", "message", "source"} {
				if _, ok := payload[field]; !ok {
					t.Errorf("log JSON missing field %q: %s", field, entry.Line)
				}
			}

			t.Logf("integration test passed — received valid JSON log from %q", entry.Source)
			return

		case <-ctx.Done():
			t.Fatal("timed out waiting for log entry from randomlog")
		}
	}
}
