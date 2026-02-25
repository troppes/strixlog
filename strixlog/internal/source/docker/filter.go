package docker

import (
	"strings"

	"github.com/docker/docker/api/types/container"
)

// isSelf returns true when the container ID starts with the given hostname.
// Inside a Docker container, HOSTNAME is set to a prefix of the container ID.
func isSelf(containerID, hostname string) bool {
	if hostname == "" {
		return false
	}
	return strings.HasPrefix(containerID, hostname)
}

// containerName extracts the primary display name for a container.
// Docker stores names with a leading slash, so we strip it.
func containerName(c container.Summary) string {
	if len(c.Names) == 0 {
		return c.ID[:12]
	}
	return strings.TrimPrefix(c.Names[0], "/")
}
