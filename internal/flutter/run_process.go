package flutter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/rs/zerolog/log"
)

// RunProcess manages a single `flutter run --machine` process.
type RunProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	mu           sync.Mutex
	appID        string
	vmServiceURI string
	running      bool
	nextID       int

	// Channels for coordination
	Started chan struct{} // closed when app.started is received
	Stopped chan struct{} // closed when process exits
	Err     error        // set if process exits with error

	// Log buffer
	logMu sync.Mutex
	logs  []LogEntry
}

type LogEntry struct {
	Message string `json:"message"`
}

// Event represents a JSON event from flutter run --machine stdout.
// Each line is a JSON array with one element: [{"event":"...","params":{...}}]
type Event struct {
	Event  string          `json:"event"`
	Params json.RawMessage `json:"params"`
}

// Response represents a JSON-RPC response from flutter run --machine.
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type appDebugPortParams struct {
	AppID string `json:"appId"`
	WsURI string `json:"wsUri"`
	Port  int    `json:"port"`
}

type appStartParams struct {
	AppID           string `json:"appId"`
	DeviceID        string `json:"deviceId"`
	SupportsRestart bool   `json:"supportsRestart"`
}

type appLogParams struct {
	AppID string `json:"appId"`
	Log   string `json:"log"`
}

// Start spawns `flutter run --machine` and begins parsing events.
func Start(flutterBin, workDir, deviceID, target string) (*RunProcess, error) {
	args := []string{"run", "--machine", "-d", deviceID}
	if target != "" {
		args = append(args, "--target", target)
	}

	cmd := exec.Command(flutterBin, args...)
	cmd.Dir = workDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start flutter run: %w", err)
	}

	p := &RunProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		Started: make(chan struct{}),
		Stopped: make(chan struct{}),
	}

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Debug().Str("flutter_stderr", scanner.Text()).Send()
		}
	}()

	// Read stdout events in background
	go p.readEvents()

	// Wait for process exit in background
	go func() {
		p.Err = cmd.Wait()
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		close(p.Stopped)
		log.Info().Err(p.Err).Msg("flutter run process exited")
	}()

	return p, nil
}

// readEvents parses JSON events from stdout line by line.
func (p *RunProcess) readEvents() {
	scanner := bufio.NewScanner(p.stdout)
	// flutter run --machine can output long lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Each line is a JSON array: [{"event":"...","params":{...}}]
		// Or a response: [{"id":1,"result":{...}}]
		var messages []json.RawMessage
		if err := json.Unmarshal(line, &messages); err != nil {
			log.Debug().Str("raw", string(line)).Msg("non-JSON line from flutter run")
			continue
		}

		for _, msg := range messages {
			p.handleMessage(msg)
		}
	}
}

func (p *RunProcess) handleMessage(raw json.RawMessage) {
	// Try as event first
	var evt Event
	if err := json.Unmarshal(raw, &evt); err == nil && evt.Event != "" {
		p.handleEvent(evt)
		return
	}

	// Try as response
	var resp Response
	if err := json.Unmarshal(raw, &resp); err == nil && resp.ID != 0 {
		p.handleResponse(resp)
		return
	}
}

func (p *RunProcess) handleEvent(evt Event) {
	switch evt.Event {
	case "app.start":
		var params appStartParams
		json.Unmarshal(evt.Params, &params)
		p.mu.Lock()
		p.appID = params.AppID
		p.mu.Unlock()
		log.Info().Str("appId", params.AppID).Msg("App starting")

	case "app.debugPort":
		var params appDebugPortParams
		json.Unmarshal(evt.Params, &params)
		p.mu.Lock()
		p.vmServiceURI = params.WsURI
		p.mu.Unlock()
		log.Info().Str("wsUri", params.WsURI).Msg("VM Service available")

	case "app.started":
		p.mu.Lock()
		p.running = true
		p.mu.Unlock()
		close(p.Started)
		log.Info().Msg("App started")

	case "app.log":
		var params appLogParams
		json.Unmarshal(evt.Params, &params)
		p.logMu.Lock()
		p.logs = append(p.logs, LogEntry{Message: params.Log})
		// Keep last 1000 entries
		if len(p.logs) > 1000 {
			p.logs = p.logs[len(p.logs)-1000:]
		}
		p.logMu.Unlock()

	case "app.stop":
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		log.Info().Msg("App stopped")

	default:
		log.Debug().Str("event", evt.Event).Msg("Unhandled flutter event")
	}
}

func (p *RunProcess) handleResponse(resp Response) {
	log.Debug().Int("id", resp.ID).Msg("Got response from flutter run")
}

// AppID returns the current app ID.
func (p *RunProcess) AppID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.appID
}

// VMServiceURI returns the VM Service WebSocket URI.
func (p *RunProcess) VMServiceURI() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vmServiceURI
}

// IsRunning returns whether the app is currently running.
func (p *RunProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// SendCommand sends a JSON-RPC command to flutter run via stdin.
func (p *RunProcess) SendCommand(method string, params map[string]any) (int, error) {
	p.mu.Lock()
	p.nextID++
	id := p.nextID
	appID := p.appID
	p.mu.Unlock()

	if params == nil {
		params = make(map[string]any)
	}
	params["appId"] = appID

	cmd := []map[string]any{
		{
			"id":     id,
			"method": method,
			"params": params,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return 0, err
	}

	data = append(data, '\n')
	if _, err := p.stdin.Write(data); err != nil {
		return 0, fmt.Errorf("failed to write to flutter stdin: %w", err)
	}

	return id, nil
}

// HotReload triggers a hot reload.
func (p *RunProcess) HotReload() error {
	_, err := p.SendCommand("app.restart", map[string]any{
		"fullRestart": false,
		"pause":       false,
	})
	return err
}

// HotRestart triggers a full restart.
func (p *RunProcess) HotRestart() error {
	_, err := p.SendCommand("app.restart", map[string]any{
		"fullRestart": true,
		"pause":       false,
	})
	return err
}

// Stop sends the stop command to flutter run.
func (p *RunProcess) Stop() error {
	_, err := p.SendCommand("app.stop", nil)
	return err
}

// Kill forcefully kills the process.
func (p *RunProcess) Kill() {
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

// Logs returns the buffered log entries.
func (p *RunProcess) Logs(last int) []LogEntry {
	p.logMu.Lock()
	defer p.logMu.Unlock()

	if last <= 0 || last > len(p.logs) {
		last = len(p.logs)
	}
	result := make([]LogEntry, last)
	copy(result, p.logs[len(p.logs)-last:])
	return result
}
