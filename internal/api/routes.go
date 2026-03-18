package api

import (
	"net/http"

	"github.com/torbenkeller/flutter-agent-connect/internal/device"
	"github.com/torbenkeller/flutter-agent-connect/internal/session"
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

	// Flutter lifecycle
	mux.HandleFunc("POST /sessions/{id}/flutter/run", h.FlutterRun)
	mux.HandleFunc("POST /sessions/{id}/flutter/stop", h.FlutterStop)
	mux.HandleFunc("POST /sessions/{id}/flutter/hot-reload", h.FlutterHotReload)
	mux.HandleFunc("POST /sessions/{id}/flutter/hot-restart", h.FlutterHotRestart)
	mux.HandleFunc("POST /sessions/{id}/flutter/clean", h.FlutterClean)
	mux.HandleFunc("POST /sessions/{id}/flutter/pub-get", h.FlutterPubGet)
	mux.HandleFunc("GET /flutter/version", h.FlutterVersion)

	// Port forwarding
	mux.HandleFunc("POST /sessions/{id}/forward", h.AddForward)
	mux.HandleFunc("GET /sessions/{id}/forward", h.ListForwards)

	// Device interaction
	mux.HandleFunc("GET /sessions/{id}/device/screenshot", h.DeviceScreenshot)
	mux.HandleFunc("POST /sessions/{id}/device/tap", h.DeviceTap)
	mux.HandleFunc("POST /sessions/{id}/device/swipe", h.DeviceSwipe)
	mux.HandleFunc("POST /sessions/{id}/device/type", h.DeviceType)

	// DevTools inspection & debugging
	mux.HandleFunc("GET /sessions/{id}/devtools/widgets", h.DevtoolsWidgets)
	mux.HandleFunc("GET /sessions/{id}/devtools/render", h.DevtoolsRender)
	mux.HandleFunc("GET /sessions/{id}/devtools/semantics", h.DevtoolsSemantics)
	mux.HandleFunc("POST /sessions/{id}/devtools/performance", h.DevtoolsPerformance)
	mux.HandleFunc("POST /sessions/{id}/devtools/paint", h.DevtoolsPaint)
	mux.HandleFunc("POST /sessions/{id}/devtools/repaint", h.DevtoolsRepaint)
	mux.HandleFunc("GET /sessions/{id}/devtools/logs", h.DevtoolsLogs)

	return mux
}
