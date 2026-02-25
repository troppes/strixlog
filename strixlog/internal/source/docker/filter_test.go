package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestIsSelf(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		hostname    string
		want        bool
	}{
		{
			name:        "full match prefix",
			containerID: "abc123def456",
			hostname:    "abc123",
			want:        true,
		},
		{
			name:        "exact match",
			containerID: "abc123",
			hostname:    "abc123",
			want:        true,
		},
		{
			name:        "no match",
			containerID: "abc123def456",
			hostname:    "xyz789",
			want:        false,
		},
		{
			name:        "empty hostname always false",
			containerID: "abc123",
			hostname:    "",
			want:        false,
		},
		{
			name:        "hostname longer than id",
			containerID: "abc",
			hostname:    "abcdef",
			want:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isSelf(tc.containerID, tc.hostname)
			if got != tc.want {
				t.Errorf("isSelf(%q, %q) = %v, want %v", tc.containerID, tc.hostname, got, tc.want)
			}
		})
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		name      string
		container container.Summary
		want      string
	}{
		{
			name:      "name with leading slash",
			container: container.Summary{ID: "abc123", Names: []string{"/mycontainer"}},
			want:      "mycontainer",
		},
		{
			name:      "name without leading slash",
			container: container.Summary{ID: "abc123", Names: []string{"mycontainer"}},
			want:      "mycontainer",
		},
		{
			name:      "no names falls back to short id",
			container: container.Summary{ID: "abc123def456789"},
			want:      "abc123def456",
		},
		{
			name:      "multiple names uses first",
			container: container.Summary{ID: "abc123", Names: []string{"/primary", "/alias"}},
			want:      "primary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containerName(tc.container)
			if got != tc.want {
				t.Errorf("containerName() = %q, want %q", got, tc.want)
			}
		})
	}
}
