package http

import (
	"context"
	"errors"
	"fmt"
	"microservice/internal/platform/logger"
	"net"
	"net/http"
	"time"

	"microservice/internal/config"
)

type Server struct {
	server *http.Server
	logger logger.Logger
}

func NewServer(cfg *config.HttpConfig, log logger.Logger, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
			IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
		},
		logger: log,
	}
}

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		s.logger.Error("failed to listen", logger.Error(err))
		return err
	}

	s.logger.Info("Starting HTTP server", logger.String("addr", s.server.Addr))

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("failed to serve", logger.Error(err))
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		s.logger.Info("Server startup cancelled")
		return s.server.Shutdown(context.Background())
	default:
		return nil
	}
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logger.Info("Shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}
