package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Mount represents a Docker container mount point.
type Mount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

// DockerInspector retrieves mount mappings for a container.
type DockerInspector interface {
	GetMounts(containerID string) ([]Mount, error)
}

// dockerCLIInspector implements DockerInspector using the docker CLI.
type dockerCLIInspector struct{}

func (d *dockerCLIInspector) GetMounts(containerID string) ([]Mount, error) {
	out, err := exec.Command("docker", "inspect", containerID, "--format", "{{json .Mounts}}").Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed for container %s: %w", containerID, err)
	}

	var mounts []Mount
	if err := json.Unmarshal(out, &mounts); err != nil {
		return nil, fmt.Errorf("failed to parse docker mounts: %w", err)
	}
	return mounts, nil
}

// resolveContainerPath translates a container-side path to a host-side path
// using Docker mount mappings. If containerID is empty, the path is returned
// unchanged (agent is not in a container).
func resolveContainerPath(inspector DockerInspector, containerID, containerPath string) (string, error) {
	if containerID == "" {
		return containerPath, nil
	}

	mounts, err := inspector.GetMounts(containerID)
	if err != nil {
		return "", err
	}

	// Find the best (longest prefix) matching mount
	var bestSource, bestDest string
	bestLen := 0

	for _, m := range mounts {
		if strings.HasPrefix(containerPath, m.Destination) && len(m.Destination) > bestLen {
			bestSource = m.Source
			bestDest = m.Destination
			bestLen = len(m.Destination)
		}
	}

	if bestLen == 0 {
		return "", fmt.Errorf("no mount found for container path %s. Available mounts: %v", containerPath, mountDestinations(mounts))
	}

	relative := strings.TrimPrefix(containerPath, bestDest)
	hostPath := filepath.Join(bestSource, relative)

	// Verify the path exists on the host
	if _, err := os.Stat(hostPath); err != nil {
		return "", fmt.Errorf("resolved path %s does not exist on host: %w", hostPath, err)
	}

	log.Debug().
		Str("container", containerPath).
		Str("host", hostPath).
		Str("mount", bestDest+" → "+bestSource).
		Msg("Resolved container path")

	return hostPath, nil
}

func mountDestinations(mounts []Mount) []string {
	result := make([]string, len(mounts))
	for i, m := range mounts {
		result[i] = m.Destination
	}
	return result
}
