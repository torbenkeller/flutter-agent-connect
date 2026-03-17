package flutter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// VMServiceClient connects to the Dart VM Service via WebSocket.
type VMServiceClient struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	nextID    atomic.Int64
	isolateID string
	ADBSerial string // set for Android emulators
}

// ConnectVMService connects to the Dart VM Service WebSocket.
func ConnectVMService(wsURI string) (*VMServiceClient, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VM Service at %s: %w", wsURI, err)
	}

	client := &VMServiceClient{conn: conn}

	// Discover the main isolate (retry — isolate may not be ready yet)
	var discoverErr error
	for i := 0; i < 5; i++ {
		if err := client.discoverIsolate(); err == nil {
			discoverErr = nil
			break
		} else {
			discoverErr = err
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	if discoverErr != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to discover isolate: %w", discoverErr)
	}

	log.Info().Str("isolate", client.isolateID).Msg("VM Service connected")
	return client, nil
}

// call sends a JSON-RPC request and reads the response synchronously.
func (c *VMServiceClient) call(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	idNum := c.nextID.Add(1)
	id := fmt.Sprintf("%d", idNum)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	if err := c.conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read messages until we get our response
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(message, &raw); err != nil {
			continue
		}

		// Check if this message has our ID
		idRaw, hasID := raw["id"]
		if !hasID {
			continue // stream notification, skip
		}

		var respID string
		if err := json.Unmarshal(idRaw, &respID); err != nil {
			// Try as number
			var idNum int64
			if err2 := json.Unmarshal(idRaw, &idNum); err2 == nil {
				respID = fmt.Sprintf("%d", idNum)
			} else {
				continue
			}
		}

		if respID != id {
			continue // not our response
		}

		// Found our response
		if errRaw, hasErr := raw["error"]; hasErr {
			return nil, fmt.Errorf("VM Service error: %s", string(errRaw))
		}

		if resultRaw, hasResult := raw["result"]; hasResult {
			return resultRaw, nil
		}

		return nil, fmt.Errorf("response has no result or error")
	}
}

// discoverIsolate finds the main Flutter isolate.
func (c *VMServiceClient) discoverIsolate() error {
	result, err := c.call("getVM", nil, 10*time.Second)
	if err != nil {
		return err
	}

	var vm struct {
		Isolates []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"isolates"`
	}
	if err := json.Unmarshal(result, &vm); err != nil {
		return fmt.Errorf("failed to parse VM info: %w", err)
	}

	if len(vm.Isolates) == 0 {
		return fmt.Errorf("no isolates found")
	}

	c.isolateID = vm.Isolates[0].ID
	return nil
}

// CallExtension calls a Flutter service extension.
func (c *VMServiceClient) CallExtension(method string, args map[string]any) (json.RawMessage, error) {
	params := map[string]any{
		"isolateId": c.isolateID,
	}
	for k, v := range args {
		params[k] = v
	}

	return c.call(method, params, 30*time.Second)
}

// EnsureSemantics forces semantics tree generation (needed on Android where
// semantics are disabled by default for performance).
// adbSerial is optional — if provided, enables accessibility via adb as fallback.
func (c *VMServiceClient) EnsureSemantics(adbSerial string) {
	// Method 1: Toggle semantics debugger (works on iOS)
	c.CallExtension("ext.flutter.showSemanticsDebugger", map[string]any{"enabled": "true"})
	time.Sleep(500 * time.Millisecond)
	c.CallExtension("ext.flutter.showSemanticsDebugger", map[string]any{"enabled": "false"})
	time.Sleep(500 * time.Millisecond)

	// Method 2: On Android, enable accessibility via adb (more reliable)
	if adbSerial != "" {
		adbPath, _ := exec.LookPath("adb")
		if adbPath == "" {
			home, _ := os.UserHomeDir()
			adbPath = filepath.Join(home, "Library", "Android", "sdk", "platform-tools", "adb")
		}
		// Enable TalkBack to force semantics generation
		exec.Command(adbPath, "-s", adbSerial, "shell",
			"settings", "put", "secure", "enabled_accessibility_services",
			"com.google.android.marvin.talkback/com.google.android.marvin.talkback.TalkBackService").Run()
		time.Sleep(2 * time.Second)
		// Disable TalkBack again (we just needed it to trigger semantics)
		exec.Command(adbPath, "-s", adbSerial, "shell",
			"settings", "put", "secure", "enabled_accessibility_services", "").Run()
		time.Sleep(500 * time.Millisecond)
	}
}

// GetSemanticsTree returns the semantics tree of the running app.
func (c *VMServiceClient) GetSemanticsTree() (*SemanticsNode, error) {
	result, err := c.CallExtension("ext.flutter.debugDumpSemanticsTreeInTraversalOrder", nil)

	// If semantics not generated, enable them and retry
	if err == nil {
		var check struct{ Data string `json:"data"` }
		json.Unmarshal(result, &check)
		if strings.Contains(check.Data, "Semantics not generated") {
			c.EnsureSemantics(c.ADBSerial)
			result, err = c.CallExtension("ext.flutter.debugDumpSemanticsTreeInTraversalOrder", nil)
		}
	}
	if err != nil {
		return nil, err
	}

	// The response has {"data": "...", "type": "_extensionType"}
	// Try "data" field first, then "value"
	var rawResult struct {
		Data  string `json:"data"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(result, &rawResult); err != nil {
		var str string
		if err2 := json.Unmarshal(result, &str); err2 != nil {
			return nil, fmt.Errorf("unexpected semantics response: %s", string(result))
		}
		return parseSemanticsText(str), nil
	}

	text := rawResult.Data
	if text == "" {
		text = rawResult.Value
	}
	if text == "" {
		return nil, fmt.Errorf("empty semantics tree response: %s", string(result)[:min(200, len(result))])
	}

	log.Debug().Int("len", len(text)).Msg("Parsing semantics tree")
	return parseSemanticsText(text), nil
}

// GetWidgetTree returns the widget tree dump.
func (c *VMServiceClient) GetWidgetTree() (string, error) {
	result, err := c.CallExtension("ext.flutter.debugDumpApp", nil)
	if err != nil {
		return "", err
	}

	var resp struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return string(result), nil
	}
	return resp.Value, nil
}

// GetRenderTree returns the render tree dump.
func (c *VMServiceClient) GetRenderTree() (string, error) {
	result, err := c.CallExtension("ext.flutter.debugDumpRenderTree", nil)
	if err != nil {
		return "", err
	}

	var resp struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return string(result), nil
	}
	return resp.Value, nil
}

// ToggleDebugFlag toggles a debug flag and returns the new state.
func (c *VMServiceClient) ToggleDebugFlag(extension string) (bool, error) {
	result, err := c.CallExtension(extension, nil)
	if err != nil {
		return false, err
	}

	var resp struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return true, nil
	}
	return resp.Enabled, nil
}

// Close closes the WebSocket connection.
func (c *VMServiceClient) Close() {
	c.conn.Close()
}

// SemanticsNode represents a node in the semantics tree.
type SemanticsNode struct {
	ID       int              `json:"id"`
	Label    string           `json:"label,omitempty"`
	Value    string           `json:"value,omitempty"`
	Hint     string           `json:"hint,omitempty"`
	Flags    []string         `json:"flags,omitempty"`
	Actions  []string         `json:"actions,omitempty"`
	Rect     *Rect            `json:"rect,omitempty"`
	Children []*SemanticsNode `json:"children,omitempty"`
}

type Rect struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
}

func (r *Rect) Center() (float64, float64) {
	return (r.Left + r.Right) / 2, (r.Top + r.Bottom) / 2
}

func (n *SemanticsNode) FindByLabel(label string, index int) *SemanticsNode {
	var matches []*SemanticsNode
	n.findByLabel(label, &matches)
	if index < len(matches) {
		return matches[index]
	}
	return nil
}

func (n *SemanticsNode) findByLabel(label string, matches *[]*SemanticsNode) {
	if n.Label != "" && containsIgnoreCase(n.Label, label) {
		*matches = append(*matches, n)
	}
	for _, child := range n.Children {
		child.findByLabel(label, matches)
	}
}

func (n *SemanticsNode) FindByKey(key string, index int) *SemanticsNode {
	var matches []*SemanticsNode
	n.findByKey(key, &matches)
	if index < len(matches) {
		return matches[index]
	}
	return nil
}

func (n *SemanticsNode) findByKey(key string, matches *[]*SemanticsNode) {
	if containsIgnoreCase(n.Value, key) || containsIgnoreCase(n.Label, key) {
		*matches = append(*matches, n)
	}
	for _, child := range n.Children {
		child.findByKey(key, matches)
	}
}

func (n *SemanticsNode) AllLabels() []string {
	var labels []string
	n.collectLabels(&labels)
	return labels
}

func (n *SemanticsNode) collectLabels(labels *[]string) {
	if n.Label != "" {
		*labels = append(*labels, n.Label)
	}
	for _, child := range n.Children {
		child.collectLabels(labels)
	}
}

func containsIgnoreCase(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	sl := toLower(s)
	sl2 := toLower(substr)
	for i := 0; i <= len(sl)-len(sl2); i++ {
		if sl[i:i+len(sl2)] == sl2 {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseSemanticsText is in semantics_parser.go
