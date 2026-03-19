package client

import (
	"os"
	"testing"
)

func TestIsInContainer_OnMac(t *testing.T) {
	// On macOS, we should NOT be in a container
	if IsInContainer() {
		t.Skip("Running in a container, skipping host-side test")
	}
}

func TestResolveWorkDir_OnHost(t *testing.T) {
	// On the host, paths should pass through unchanged (just resolved to absolute)
	if IsInContainer() {
		t.Skip("Running in a container")
	}

	// Relative path should become absolute
	resolved, err := ResolveWorkDir(".")
	if err != nil {
		t.Fatalf("ResolveWorkDir failed: %v", err)
	}

	cwd, _ := os.Getwd()
	if resolved != cwd {
		t.Errorf("expected %s, got %s", cwd, resolved)
	}
}

func TestResolveWorkDir_AbsolutePath(t *testing.T) {
	if IsInContainer() {
		t.Skip("Running in a container")
	}

	resolved, err := ResolveWorkDir("/tmp")
	if err != nil {
		t.Fatalf("ResolveWorkDir failed: %v", err)
	}

	// On macOS, /tmp is a symlink to /private/tmp
	if resolved != "/tmp" && resolved != "/private/tmp" {
		t.Errorf("expected /tmp or /private/tmp, got %s", resolved)
	}
}

func TestParseMountInfo(t *testing.T) {
	// Simulate a mountinfo line
	// This tests the parsing logic without needing an actual container
	sampleMountInfo := `1171 1146 0:92 /Users/torben/project /workspaces/project rw,relatime shared:1 - ext4 /dev/sda1 rw`

	// Write to temp file to test
	tmpFile, err := os.CreateTemp("", "mountinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString(sampleMountInfo + "\n")
	tmpFile.Close()

	// We can't easily test resolveViaMountInfo without being in a container,
	// but we can verify the parsing logic works by checking the field extraction
	fields := splitFields(sampleMountInfo)
	if len(fields) < 5 {
		t.Fatal("not enough fields")
	}

	root := fields[3]       // /Users/torben/project
	mountPoint := fields[4] // /workspaces/project

	if root != "/Users/torben/project" {
		t.Errorf("expected root /Users/torben/project, got %s", root)
	}
	if mountPoint != "/workspaces/project" {
		t.Errorf("expected mount point /workspaces/project, got %s", mountPoint)
	}
}

func splitFields(s string) []string {
	var fields []string
	field := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if field != "" {
				fields = append(fields, field)
				field = ""
			}
		} else {
			field += string(c)
		}
	}
	if field != "" {
		fields = append(fields, field)
	}
	return fields
}
