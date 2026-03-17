package api

import (
	"encoding/json"
	"net/http"

	"github.com/torben/flutter-agent-connect/internal/device"
	"github.com/torben/flutter-agent-connect/internal/session"
	"github.com/torben/flutter-agent-connect/pkg/models"
)

type Handlers struct {
	sessions *session.Manager
	devices  *device.Pool
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "0.1.0",
	})
}

func (h *Handlers) ListDevices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"devices": h.devices.List(),
	})
}

func (h *Handlers) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing required field: id")
		return
	}

	agent := h.sessions.RegisterAgent(req.ID)
	writeJSON(w, http.StatusCreated, agent)
}

func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing X-Agent-ID header")
		return
	}

	var req struct {
		Platform   string `json:"platform"`
		DeviceType string `json:"device_type"`
		Name       string `json:"name"`
		WorkDir    string `json:"work_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	if req.Platform == "" {
		req.Platform = "ios"
	}

	platform := models.PlatformType(req.Platform)
	if !platform.IsValid() {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid platform: "+req.Platform)
		return
	}

	s, err := h.sessions.CreateSession(agentID, platform, req.DeviceType, req.Name, req.WorkDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing X-Agent-ID header")
		return
	}

	sessions := h.sessions.ListSessions(agentID)
	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
	})
}

func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	s, err := h.sessions.GetSession(agentID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s)
}

func (h *Handlers) DeleteSession(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	if err := h.sessions.DestroySession(agentID, id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Session destroyed"})
}

// Flutter lifecycle handlers

func (h *Handlers) FlutterRun(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	var req struct {
		Target string `json:"target"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Target == "" {
		req.Target = "lib/main.dart"
	}

	result, err := h.sessions.StartApp(agentID, id, req.Target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) FlutterStop(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	if err := h.sessions.StopApp(agentID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "App stopped"})
}

func (h *Handlers) FlutterHotReload(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	if err := h.sessions.HotReload(agentID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (h *Handlers) FlutterHotRestart(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	if err := h.sessions.HotRestart(agentID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (h *Handlers) FlutterClean(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	result, err := h.sessions.FlutterClean(agentID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) FlutterPubGet(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	result, err := h.sessions.FlutterPubGet(agentID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) FlutterVersion(w http.ResponseWriter, r *http.Request) {
	version, err := h.sessions.FlutterVersion()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, version)
}

// Device interaction handlers

func (h *Handlers) DeviceScreenshot(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	data, err := h.sessions.Screenshot(agentID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handlers) DeviceTap(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	var req struct {
		Label string  `json:"label"`
		Key   string  `json:"key"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		Index int     `json:"index"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := h.sessions.DeviceTap(agentID, id, req.Label, req.Key, req.X, req.Y, req.Index)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) DeviceSwipe(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	var req struct {
		Direction  string `json:"direction"`
		DurationMs int    `json:"duration_ms"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.DurationMs == 0 {
		req.DurationMs = 300
	}

	if err := h.sessions.DeviceSwipe(agentID, id, req.Direction, req.DurationMs); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handlers) DeviceType(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	var req struct {
		Text  string `json:"text"`
		Clear bool   `json:"clear"`
		Enter bool   `json:"enter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Missing required field: text")
		return
	}

	if err := h.sessions.DeviceType(agentID, id, req.Text, req.Clear, req.Enter); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "text_entered": req.Text})
}

// DevTools inspection & debugging handlers

func (h *Handlers) DevtoolsWidgets(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsRender(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsSemantics(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsPerformance(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsPaint(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsRepaint(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DevtoolsLogs(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
