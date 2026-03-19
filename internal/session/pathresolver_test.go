package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockDockerInspector struct {
	mounts []Mount
	err    error
}

func (m *mockDockerInspector) GetMounts(_ string) ([]Mount, error) {
	return m.mounts, m.err
}

func TestResolveContainerPath_NoContainer(t *testing.T) {
	// Empty container ID = not in a container, path passes through
	result, err := resolveContainerPath(&mockDockerInspector{}, "", "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/some/path" {
		t.Errorf("expected /some/path, got %s", result)
	}
}

func TestResolveContainerPath_ExactMount(t *testing.T) {
	tmpDir := t.TempDir()

	inspector := &mockDockerInspector{
		mounts: []Mount{
			{Source: tmpDir, Destination: "/workspaces/project"},
		},
	}

	result, err := resolveContainerPath(inspector, "abc123", "/workspaces/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != tmpDir {
		t.Errorf("expected %s, got %s", tmpDir, result)
	}
}

func TestResolveContainerPath_SubDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "lib")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	inspector := &mockDockerInspector{
		mounts: []Mount{
			{Source: tmpDir, Destination: "/workspaces/project"},
		},
	}

	result, err := resolveContainerPath(inspector, "abc123", "/workspaces/project/lib")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != subDir {
		t.Errorf("expected %s, got %s", subDir, result)
	}
}

func TestResolveContainerPath_LongestPrefixMatch(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "inner")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	inspector := &mockDockerInspector{
		mounts: []Mount{
			{Source: "/wrong/path", Destination: "/workspaces"},
			{Source: tmpDir, Destination: "/workspaces/project"},
		},
	}

	result, err := resolveContainerPath(inspector, "abc123", "/workspaces/project/inner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nestedDir {
		t.Errorf("expected %s, got %s", nestedDir, result)
	}
}

func TestResolveContainerPath_NoMatchingMount(t *testing.T) {
	inspector := &mockDockerInspector{
		mounts: []Mount{
			{Source: "/Users/torben/other", Destination: "/workspaces/other"},
		},
	}

	_, err := resolveContainerPath(inspector, "abc123", "/app/project")
	if err == nil {
		t.Error("expected error for no matching mount")
	}
}

func TestResolveContainerPath_ResolvedPathNotExist(t *testing.T) {
	inspector := &mockDockerInspector{
		mounts: []Mount{
			{Source: "/nonexistent/host/path", Destination: "/workspaces/project"},
		},
	}

	_, err := resolveContainerPath(inspector, "abc123", "/workspaces/project")
	if err == nil {
		t.Error("expected error when resolved path does not exist")
	}
}

func TestResolveContainerPath_DockerInspectFailure(t *testing.T) {
	inspector := &mockDockerInspector{
		err: fmt.Errorf("docker not available"),
	}

	_, err := resolveContainerPath(inspector, "abc123", "/workspaces/project")
	if err == nil {
		t.Error("expected error when docker inspect fails")
	}
}
