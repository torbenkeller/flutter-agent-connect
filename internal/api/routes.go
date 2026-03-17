package api

import (
	"net/http"

	"github.com/torben/flutter-agent-connect/internal/device"
	"github.com/torben/flutter-agent-connect/internal/session"
)

func NewRouter(mgr *session.Manager, pool *device.Pool) *http.ServeMux {
	h := &Handlers{
		sessions: mgr,
		devices:  pool,
	}

	mux := http.NewServeMux()

	// Infra
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /devices", h.ListDevices)

	// Agents
	mux.HandleFunc("POST /agents", h.RegisterAgent)

	// Sessions
	mux.HandleFunc("POST /sessions", h.CreateSession)
	mux.HandleFunc("GET /sessions", h.ListSessions)
	mux.HandleFunc("GET /sessions/{id}", h.GetSession)
	mux.HandleFunc("DELETE /sessions/{id}", h.DeleteSession)

	// App lifecycle
	mux.HandleFunc("POST /sessions/{id}/app/start", h.AppStart)
	mux.HandleFunc("POST /sessions/{id}/app/reload", h.AppReload)
	mux.HandleFunc("POST /sessions/{id}/app/restart", h.AppRestart)
	mux.HandleFunc("POST /sessions/{id}/app/stop", h.AppStop)

	// Screenshots & UI
	mux.HandleFunc("GET /sessions/{id}/screenshot", h.Screenshot)
	mux.HandleFunc("POST /sessions/{id}/tap", h.Tap)
	mux.HandleFunc("POST /sessions/{id}/swipe", h.Swipe)
	mux.HandleFunc("POST /sessions/{id}/type", h.TypeText)

	// Inspection
	mux.HandleFunc("GET /sessions/{id}/inspect/widgets", h.InspectWidgets)
	mux.HandleFunc("GET /sessions/{id}/inspect/render", h.InspectRender)
	mux.HandleFunc("GET /sessions/{id}/inspect/semantics", h.InspectSemantics)

	// Debugging
	mux.HandleFunc("POST /sessions/{id}/debug/paint", h.DebugPaint)
	mux.HandleFunc("POST /sessions/{id}/debug/repaint", h.DebugRepaint)
	mux.HandleFunc("POST /sessions/{id}/debug/performance", h.DebugPerformance)

	// Logs
	mux.HandleFunc("GET /sessions/{id}/logs", h.GetLogs)

	return mux
}
