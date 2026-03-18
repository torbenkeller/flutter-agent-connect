package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient creates a client pointing at the given httptest server.
func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		ServerURL:       srv.URL,
		AgentID:         "test-agent",
		ActiveSessionID: "s1",
		http:            srv.Client(),
	}
}

// simpleMux creates a mux with preset handlers for testing.
func simpleMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})

	mux.HandleFunc("GET /sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessions": []map[string]any{
				{"id": "s1", "name": "ios", "platform": "ios"},
				{"id": "s2", "name": "android", "platform": "android"},
			},
		})
	})

	mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "notfound" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found", "message": "session not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "name": "test", "platform": "ios"})
	})

	mux.HandleFunc("DELETE /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Session destroyed"})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/run", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req["target"] == "broken.dart" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":        "build_error",
				"message":      "build failed",
				"build_output": []string{"lib/main.dart:4: Error: Expected ';'"},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"app_id": "app-123",
			"state":  "running",
		})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/stop", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "App stopped"})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/hot-reload", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "reload_duration_ms": 150})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/hot-restart", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "reload_duration_ms": 2500})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/clean", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "flutter clean completed"})
	})

	mux.HandleFunc("POST /sessions/{id}/flutter/pub-get", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "flutter pub get completed"})
	})

	mux.HandleFunc("GET /flutter/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"frameworkVersion": "3.24.0", "channel": "stable"})
	})

	mux.HandleFunc("GET /sessions/{id}/device/screenshot", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG magic
	})

	mux.HandleFunc("POST /sessions/{id}/device/tap", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"x":       150,
			"y":       225,
			"element": req["label"],
		})
	})

	mux.HandleFunc("POST /sessions/{id}/device/swipe", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	mux.HandleFunc("POST /sessions/{id}/device/type", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["text"] == nil || req["text"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "validation_error", "message": "Missing text"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	mux.HandleFunc("GET /sessions/{id}/devtools/widgets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "widgets", "data": "MyApp\n └Scaffold"})
	})

	mux.HandleFunc("GET /sessions/{id}/devtools/render", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "render", "data": "RenderView#abc"})
	})

	mux.HandleFunc("GET /sessions/{id}/devtools/semantics", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "semantics", "data": map[string]any{"id": 0, "label": "root"}})
	})

	mux.HandleFunc("POST /sessions/{id}/devtools/paint", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"flag": "paint", "enabled": true})
	})

	mux.HandleFunc("GET /sessions/{id}/devtools/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "line1")
		fmt.Fprintln(w, "line2")
		fmt.Fprintln(w, "line3")
	})

	mux.HandleFunc("POST /sessions/{id}/forward", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"container_port": 8080,
			"host_port":      9001,
			"env_name":       "API_URL",
			"url_ios":        "http://localhost:9001",
			"url_android":    "http://10.0.2.2:9001",
		})
	})

	mux.HandleFunc("GET /sessions/{id}/forward", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"forwards": []map[string]any{
				{"container_port": 8080, "host_port": 9001, "env_name": "API_URL"},
			},
		})
	})

	return mux
}

// --- Tests ---

func TestClientHealth(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	h, err := c.Health()
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", h.Status)
	}
	if h.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got '%s'", h.Version)
	}
}

func TestClientListSessions(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestClientGetSession(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	s, err := c.GetSession("s1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if s.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", s.Name)
	}
}

func TestClientGetSessionNotFound(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	_, err := c.GetSession("notfound")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestClientDestroySession(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	if err := c.DestroySession("s1"); err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}
}

func TestClientFlutterRun(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.FlutterRun("", "lib/main.dart")
	if err != nil {
		t.Fatalf("FlutterRun failed: %v", err)
	}
	if result.State != "running" {
		t.Errorf("expected state 'running', got '%s'", result.State)
	}
	if result.AppID != "app-123" {
		t.Errorf("expected app ID 'app-123', got '%s'", result.AppID)
	}
}

func TestClientFlutterRunBuildError(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	_, err := c.FlutterRun("", "broken.dart")
	if err == nil {
		t.Fatal("expected build error")
	}

	buildErr, ok := err.(*BuildError)
	if !ok {
		t.Fatalf("expected *BuildError, got %T: %v", err, err)
	}
	if len(buildErr.BuildOutput) != 1 {
		t.Errorf("expected 1 build output line, got %d", len(buildErr.BuildOutput))
	}
	if buildErr.BuildOutput[0] != "lib/main.dart:4: Error: Expected ';'" {
		t.Errorf("unexpected build output: %s", buildErr.BuildOutput[0])
	}
}

func TestClientFlutterRunNoSession(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)
	c.ActiveSessionID = ""

	_, err := c.FlutterRun("", "lib/main.dart")
	if err == nil {
		t.Error("expected error when no active session")
	}
}

func TestClientFlutterStop(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	if err := c.FlutterStop(""); err != nil {
		t.Fatalf("FlutterStop failed: %v", err)
	}
}

func TestClientFlutterHotReload(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.FlutterHotReload("")
	if err != nil {
		t.Fatalf("FlutterHotReload failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.DurationMs != 150 {
		t.Errorf("expected 150ms, got %d", result.DurationMs)
	}
}

func TestClientFlutterHotRestart(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.FlutterHotRestart("")
	if err != nil {
		t.Fatalf("FlutterHotRestart failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestClientFlutterClean(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.FlutterClean("")
	if err != nil {
		t.Fatalf("FlutterClean failed: %v", err)
	}
	if result.Message != "flutter clean completed" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestClientFlutterPubGet(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.FlutterPubGet("")
	if err != nil {
		t.Fatalf("FlutterPubGet failed: %v", err)
	}
	if result.Message != "flutter pub get completed" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestClientFlutterVersion(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	version, err := c.FlutterVersion()
	if err != nil {
		t.Fatalf("FlutterVersion failed: %v", err)
	}
	if version == "" {
		t.Error("expected non-empty version")
	}
}

func TestClientScreenshot(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	data, err := c.DeviceScreenshot("", false)
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}
	if len(data) != 4 {
		t.Errorf("expected 4 bytes (PNG magic), got %d", len(data))
	}
	if data[0] != 0x89 || data[1] != 0x50 {
		t.Error("unexpected PNG magic bytes")
	}
}

func TestClientTapByLabel(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.TapByLabel("", "Login", 0)
	if err != nil {
		t.Fatalf("TapByLabel failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Element != "Login" {
		t.Errorf("expected element 'Login', got '%s'", result.Element)
	}
}

func TestClientTapByCoordinates(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	result, err := c.TapAtCoordinates("", 100, 200)
	if err != nil {
		t.Fatalf("TapAtCoordinates failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestClientSwipe(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	if err := c.Swipe("", "down", 300); err != nil {
		t.Fatalf("Swipe failed: %v", err)
	}
}

func TestClientTypeText(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	if err := c.TypeText("", "hello@test.com", true, true); err != nil {
		t.Fatalf("TypeText failed: %v", err)
	}
}

func TestClientInspect(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	tests := []struct {
		treeType string
		checkKey string
	}{
		{"widgets", "type"},
		{"render", "type"},
		{"semantics", "type"},
	}

	for _, tt := range tests {
		t.Run(tt.treeType, func(t *testing.T) {
			result, err := c.Inspect("", tt.treeType)
			if err != nil {
				t.Fatalf("Inspect(%s) failed: %v", tt.treeType, err)
			}
			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("expected map result, got %T", result)
			}
			if resultMap["type"] != tt.treeType {
				t.Errorf("expected type '%s', got '%v'", tt.treeType, resultMap["type"])
			}
		})
	}
}

func TestClientToggleDebug(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	enabled, err := c.ToggleDebug("", "paint")
	if err != nil {
		t.Fatalf("ToggleDebug failed: %v", err)
	}
	if !enabled {
		t.Error("expected enabled=true")
	}
}

func TestClientGetLogs(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	logs, err := c.GetLogs("", 0)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 log lines, got %d", len(logs))
	}
	if logs[0] != "line1" {
		t.Errorf("expected 'line1', got '%s'", logs[0])
	}
}

func TestClientAddForward(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	fwd, err := c.AddForward("", 8080, "API_URL")
	if err != nil {
		t.Fatalf("AddForward failed: %v", err)
	}
	if fwd.HostPort != 9001 {
		t.Errorf("expected host port 9001, got %d", fwd.HostPort)
	}
	if fwd.EnvName != "API_URL" {
		t.Errorf("expected env name 'API_URL', got '%s'", fwd.EnvName)
	}
}

func TestClientListForwards(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	forwards, err := c.ListForwards("")
	if err != nil {
		t.Fatalf("ListForwards failed: %v", err)
	}
	if len(forwards) != 1 {
		t.Errorf("expected 1 forward, got %d", len(forwards))
	}
}

func TestClientResolveSession(t *testing.T) {
	srv := httptest.NewServer(simpleMux())
	defer srv.Close()
	c := newTestClient(srv)

	// Explicit session ID overrides active
	if c.resolveSession("explicit") != "explicit" {
		t.Error("explicit session should be returned")
	}

	// Empty falls back to active
	if c.resolveSession("") != "s1" {
		t.Errorf("expected active session 's1', got '%s'", c.resolveSession(""))
	}

	// No active session
	c.ActiveSessionID = ""
	if c.resolveSession("") != "" {
		t.Error("should return empty when no active session")
	}
}

func TestClientServerDown(t *testing.T) {
	// Server that's immediately closed
	srv := httptest.NewServer(simpleMux())
	srv.Close()
	c := newTestClient(srv)

	_, err := c.Health()
	if err == nil {
		t.Error("expected error when server is down")
	}
}
