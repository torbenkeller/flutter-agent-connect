package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/torbenkeller/flutter-agent-connect/internal/api"
	"github.com/torbenkeller/flutter-agent-connect/internal/device"
	"github.com/torbenkeller/flutter-agent-connect/internal/session"
)

type Server struct {
	config         Config
	sessionManager *session.Manager
	devicePool     *device.Pool
	httpServer     *http.Server
}

func New(cfg Config) (*Server, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	pool := device.NewPool()
	mgr := session.NewManager(pool, cfg.FlutterSDK)

	mux := api.NewRouter(mgr, pool)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		config:         cfg,
		sessionManager: mgr,
		devicePool:     pool,
		httpServer:     srv,
	}, nil
}

func (s *Server) Run() error {
	// Discover available devices
	count, err := s.devicePool.Discover()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to discover devices")
	} else {
		log.Info().Int("count", count).Msg("Discovered simulators")
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", s.httpServer.Addr).Msg("FAC Server starting")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	<-stop
	log.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.sessionManager.DestroyAll()

	return s.httpServer.Shutdown(ctx)
}
