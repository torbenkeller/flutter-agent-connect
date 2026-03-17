package flutter

import (
	"encoding/json"
	"testing"
)

func TestParseAppStartEvent(t *testing.T) {
	line := `[{"event":"app.start","params":{"appId":"abc-123","deviceId":"D59AF284","supportsRestart":true,"launchMode":"run"}}]`

	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(line), &messages); err != nil {
		t.Fatalf("failed to parse line: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	var evt Event
	if err := json.Unmarshal(messages[0], &evt); err != nil {
		t.Fatalf("failed to parse event: %v", err)
	}

	if evt.Event != "app.start" {
		t.Errorf("expected event 'app.start', got '%s'", evt.Event)
	}

	var params appStartParams
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		t.Fatalf("failed to parse params: %v", err)
	}

	if params.AppID != "abc-123" {
		t.Errorf("expected appId 'abc-123', got '%s'", params.AppID)
	}
	if params.DeviceID != "D59AF284" {
		t.Errorf("expected deviceId 'D59AF284', got '%s'", params.DeviceID)
	}
	if !params.SupportsRestart {
		t.Error("expected supportsRestart true")
	}
}

func TestParseDebugPortEvent(t *testing.T) {
	line := `[{"event":"app.debugPort","params":{"appId":"abc-123","port":52981,"wsUri":"ws://127.0.0.1:52981/Agp3e_ySUH0=/ws"}}]`

	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(line), &messages); err != nil {
		t.Fatalf("failed to parse line: %v", err)
	}

	var evt Event
	json.Unmarshal(messages[0], &evt)

	if evt.Event != "app.debugPort" {
		t.Errorf("expected 'app.debugPort', got '%s'", evt.Event)
	}

	var params appDebugPortParams
	json.Unmarshal(evt.Params, &params)

	if params.WsURI != "ws://127.0.0.1:52981/Agp3e_ySUH0=/ws" {
		t.Errorf("unexpected wsUri: %s", params.WsURI)
	}
	if params.Port != 52981 {
		t.Errorf("expected port 52981, got %d", params.Port)
	}
}

func TestParseAppLogEvent(t *testing.T) {
	line := `[{"event":"app.log","params":{"appId":"abc-123","log":"flutter: Hello World"}}]`

	var messages []json.RawMessage
	json.Unmarshal([]byte(line), &messages)

	var evt Event
	json.Unmarshal(messages[0], &evt)

	if evt.Event != "app.log" {
		t.Errorf("expected 'app.log', got '%s'", evt.Event)
	}

	var params appLogParams
	json.Unmarshal(evt.Params, &params)

	if params.Log != "flutter: Hello World" {
		t.Errorf("unexpected log: %s", params.Log)
	}
}

func TestParseResponse(t *testing.T) {
	line := `[{"id":1,"result":{"code":0,"message":"Success"}}]`

	var messages []json.RawMessage
	json.Unmarshal([]byte(line), &messages)

	var resp Response
	if err := json.Unmarshal(messages[0], &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}
}

func TestParseNonJSONLine(t *testing.T) {
	line := `Launching lib/main.dart on iPhone 16 Pro in debug mode...`

	var messages []json.RawMessage
	err := json.Unmarshal([]byte(line), &messages)

	// Should fail to parse - this is expected
	if err == nil {
		t.Error("expected parse error for non-JSON line")
	}
}

func TestBuildCommand(t *testing.T) {
	appID := "test-app-123"
	method := "app.restart"
	params := map[string]any{
		"appId":       appID,
		"fullRestart": false,
		"pause":       false,
	}

	cmd := []map[string]any{
		{
			"id":     1,
			"method": method,
			"params": params,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	// Verify it's valid JSON array
	var parsed []json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("command is not valid JSON array: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("expected 1 command, got %d", len(parsed))
	}

	// Verify structure
	var cmdMap map[string]any
	json.Unmarshal(parsed[0], &cmdMap)

	if cmdMap["method"] != "app.restart" {
		t.Errorf("expected method 'app.restart', got '%v'", cmdMap["method"])
	}
}
