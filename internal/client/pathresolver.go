package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveWorkDir resolves a work directory path to a host-side path.
// If running inside a Docker container, it translates the container path
// to the corresponding host mount path.
func ResolveWorkDir(path string) (string, error) {
	// Resolve to absolute path first
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if we're inside a Docker container
	if !isInContainer() {
		// Running directly on host — path is already correct
		return absPath, nil
	}

	// We're in a container — translate container path to host path
	hostPath, err := containerToHostPath(absPath)
	if err != nil {
		return "", fmt.Errorf("could not translate container path %q to host path: %w\n"+
			"Hint: specify the Mac-side path explicitly with --work-dir /Users/.../project", absPath, err)
	}

	return hostPath, nil
}

// isInContainer checks if we're running inside a Docker container.
func isInContainer() bool {
	// Method 1: Check /.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check /proc/1/cgroup for docker/containerd
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// Method 3: Check for container env vars
	if os.Getenv("REMOTE_CONTAINERS") != "" || os.Getenv("CODESPACES") != "" {
		return true
	}

	return false
}

// containerToHostPath translates a container path to the host path using mount info.
func containerToHostPath(containerPath string) (string, error) {
	// Method 1: Parse /proc/self/mountinfo
	hostPath, err := resolveViaMountInfo(containerPath)
	if err == nil {
		return hostPath, nil
	}

	// Method 2: Use docker inspect on our own container
	hostPath, err = resolveViaDockerInspect(containerPath)
	if err == nil {
		return hostPath, nil
	}

	return "", fmt.Errorf("no mount mapping found for %s", containerPath)
}

// resolveViaMountInfo reads /proc/self/mountinfo to find bind mounts.
// Lines look like:
// 1171 1146 0:92 /Users/torben/project /workspaces/project rw,relatime ...
func resolveViaMountInfo(containerPath string) (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	defer f.Close()

	type mountEntry struct {
		hostPath      string
		containerPath string
	}

	var mounts []mountEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}

		// Fields: mount_id parent_id major:minor root mount_point ...
		root := fields[3]       // host-side path (source)
		mountPoint := fields[4] // container-side path (target)

		// Skip non-bind mounts (proc, sys, dev, etc.)
		if strings.HasPrefix(mountPoint, "/proc") ||
			strings.HasPrefix(mountPoint, "/sys") ||
			strings.HasPrefix(mountPoint, "/dev") ||
			mountPoint == "/" {
			continue
		}

		mounts = append(mounts, mountEntry{
			hostPath:      root,
			containerPath: mountPoint,
		})
	}

	// Find the best (longest) matching mount for our path
	var bestMatch mountEntry
	bestLen := 0

	for _, m := range mounts {
		if strings.HasPrefix(containerPath, m.containerPath) && len(m.containerPath) > bestLen {
			bestMatch = m
			bestLen = len(m.containerPath)
		}
	}

	if bestLen == 0 {
		return "", fmt.Errorf("no mount found for %s", containerPath)
	}

	// Translate: replace the container mount point with the host path
	relative := strings.TrimPrefix(containerPath, bestMatch.containerPath)
	hostPath := filepath.Join(bestMatch.hostPath, relative)

	// Docker Desktop for Mac: VirtioFS mounts host paths relative to /Users,
	// so mountinfo shows "/torben/project" instead of "/Users/torben/project".
	// Detect and fix this.
	if !strings.HasPrefix(hostPath, "/Users/") && !strings.HasPrefix(hostPath, "/home/") && !strings.HasPrefix(hostPath, "/tmp/") {
		withUsers := "/Users" + hostPath
		// Check if this looks like a valid macOS path (has at least /Users/<username>/...)
		parts := strings.SplitN(strings.TrimPrefix(withUsers, "/Users/"), "/", 2)
		if len(parts) >= 1 && parts[0] != "" {
			hostPath = withUsers
		}
	}

	return hostPath, nil
}

// resolveViaDockerInspect uses `docker inspect` to find mount mappings.
func resolveViaDockerInspect(containerPath string) (string, error) {
	// Get our container ID
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// Docker containers typically have the container ID as hostname
	out, err := exec.Command("docker", "inspect", hostname, "--format", "{{json .Mounts}}").Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %w", err)
	}

	var mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Type        string `json:"Type"`
	}
	if err := json.Unmarshal(out, &mounts); err != nil {
		return "", err
	}

	// Find the best matching mount
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
		return "", fmt.Errorf("no mount found for %s", containerPath)
	}

	relative := strings.TrimPrefix(containerPath, bestDest)
	return filepath.Join(bestSource, relative), nil
}
