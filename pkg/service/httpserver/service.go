package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type service struct {
	httpServer *http.Server
}

func NewService(listenAddr string, router http.Handler) (*service, error) {
	return &service{
		httpServer: &http.Server{
			Addr:              listenAddr,
			Handler:           router,
			ReadHeaderTimeout: 3 * time.Second,
		},
	}, nil
}

func (s *service) Name() string { return "http-server" }

func (s *service) Run(ctx context.Context) error {
	slog.Info("starting http-server service", "addr", s.httpServer.Addr)
	defer slog.Info("stopped http-server service")

	errCh := make(chan error, 1)
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http-server unexpectedly closed: %v", err)
		}
	case <-ctx.Done():
		if err := s.shutdownGracefully(60 * time.Second); err != nil {
			return fmt.Errorf("shutting down http-server: %v", err)
		}
		<-doneCh
	}
	return nil
}

func (s *service) shutdownGracefully(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
