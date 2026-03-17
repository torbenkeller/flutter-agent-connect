package flutter

import (
	"encoding/json"
	"fmt"
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
	pending   map[int64]chan json.RawMessage
	pendingMu sync.Mutex
	isolateID string
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// ConnectVMService connects to the Dart VM Service WebSocket.
func ConnectVMService(wsURI string) (*VMServiceClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VM Service at %s: %w", wsURI, err)
	}

	client := &VMServiceClient{
		conn:    conn,
		pending: make(map[int64]chan json.RawMessage),
	}

	// Read responses in background
	go client.readLoop()

	// Discover the main isolate
	if err := client.discoverIsolate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to discover isolate: %w", err)
	}

	log.Info().Str("isolate", client.isolateID).Msg("VM Service connected")
	return client, nil
}

// readLoop reads WebSocket messages and dispatches responses.
func (c *VMServiceClient) readLoop() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Debug().Err(err).Msg("VM Service WebSocket read error")
			return
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		if resp.ID != 0 {
			c.pendingMu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.pendingMu.Unlock()

			if ok {
				if resp.Error != nil {
					ch <- resp.Error
				} else {
					ch <- resp.Result
				}
			}
		}
	}
}

// call sends a JSON-RPC request and waits for the response.
func (c *VMServiceClient) call(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	c.mu.Lock()
	err := c.conn.WriteJSON(req)
	c.mu.Unlock()

	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(timeout):
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
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

	// Use the first isolate (usually the main one)
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

// GetSemanticsTree returns the semantics tree of the running app.
func (c *VMServiceClient) GetSemanticsTree() (*SemanticsNode, error) {
	return c.getStructuredSemantics()
}

// getStructuredSemantics gets the semantics tree in a structured format.
func (c *VMServiceClient) getStructuredSemantics() (*SemanticsNode, error) {
	// First, ensure semantics are enabled
	c.CallExtension("ext.flutter.debugDumpSemanticsTreeInTraversalOrder", nil)

	// Get the root semantics node via the inspector
	result, err := c.CallExtension("ext.flutter.debugDumpSemanticsTreeInTraversalOrder", map[string]any{
		"inverseClip": "true",
	})
	if err != nil {
		return nil, err
	}

	// The result is a formatted string - parse it into our structure
	var rawResult struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(result, &rawResult); err != nil {
		// Try parsing as direct string
		var str string
		if err2 := json.Unmarshal(result, &str); err2 != nil {
			return nil, fmt.Errorf("unexpected semantics response: %s", string(result))
		}
		return parseSemanticsText(str), nil
	}

	return parseSemanticsText(rawResult.Value), nil
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
		var str string
		if err2 := json.Unmarshal(result, &str); err2 == nil {
			return str, nil
		}
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
		var str string
		if err2 := json.Unmarshal(result, &str); err2 == nil {
			return str, nil
		}
		return string(result), nil
	}
	return resp.Value, nil
}

// ToggleDebugFlag toggles a debug flag and returns the new state.
func (c *VMServiceClient) ToggleDebugFlag(extension string) (bool, error) {
	// These extensions toggle on each call
	result, err := c.CallExtension(extension, nil)
	if err != nil {
		return false, err
	}

	var resp struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		// Some extensions return differently
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

// Rect represents a bounding rectangle.
type Rect struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
}

// Center returns the center point of the rect.
func (r *Rect) Center() (float64, float64) {
	return (r.Left + r.Right) / 2, (r.Top + r.Bottom) / 2
}

// FindByLabel searches the semantics tree for a node with the given label.
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

// FindByKey searches for a node whose value/label contains the key identifier.
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

// AllLabels returns all labels in the tree (for error messages).
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
	return len(s) > 0 && len(substr) > 0 &&
		contains(toLower(s), toLower(substr))
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

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// parseSemanticsText is a placeholder that creates a basic tree from the debug dump.
// TODO: implement proper parsing of the Flutter semantics debug dump format.
func parseSemanticsText(text string) *SemanticsNode {
	return &SemanticsNode{
		ID:    0,
		Label: "root",
		Value: text,
	}
}
