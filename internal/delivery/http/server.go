package http

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tentens-tech/shared-lock/internal/application"
	"github.com/tentens-tech/shared-lock/internal/config"
)

type Server struct {
	app    *application.Application
	Server *http.Server
}

func New(app *application.Application) *Server {
	return &Server{
		app: app,
	}
}

func (s *Server) Start(cfg *config.ServerCfg) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/lease", s.handleLease)
	mux.HandleFunc("/keepalive", s.handleKeepalive)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.Server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Timeout.Read,
		WriteTimeout: cfg.Timeout.Write,
		IdleTimeout:  cfg.Timeout.Idle,
	}

	return s.Server.ListenAndServe()
}

func (s *Server) handleLease(w http.ResponseWriter, r *http.Request) {
	application.GetLeaseHandler(s.app.Ctx, s.app.Config, s.app.LeaseCache)(w, r)
}

func (s *Server) handleKeepalive(w http.ResponseWriter, r *http.Request) {
	application.KeepaliveHandler(s.app.Ctx, s.app.Config)(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	application.HealthHandler()(w, r)
}
