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

func (h *Handlers) AppStart(w http.ResponseWriter, r *http.Request) {
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

func (h *Handlers) AppReload(w http.ResponseWriter, r *http.Request) {
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

func (h *Handlers) AppRestart(w http.ResponseWriter, r *http.Request) {
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

func (h *Handlers) AppStop(w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	id := r.PathValue("id")

	if err := h.sessions.StopApp(agentID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "App stopped"})
}
func (h *Handlers) Screenshot(w http.ResponseWriter, r *http.Request) {
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
func (h *Handlers) Tap(w http.ResponseWriter, r *http.Request)        { writeError(w, http.StatusNotImplemented, "not_implemented", "TODO") }
func (h *Handlers) Swipe(w http.ResponseWriter, r *http.Request)      { writeError(w, http.StatusNotImplemented, "not_implemented", "TODO") }
func (h *Handlers) TypeText(w http.ResponseWriter, r *http.Request)   { writeError(w, http.StatusNotImplemented, "not_implemented", "TODO") }
func (h *Handlers) InspectWidgets(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) InspectRender(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) InspectSemantics(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DebugPaint(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DebugRepaint(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) DebugPerformance(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
func (h *Handlers) GetLogs(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "TODO")
}
