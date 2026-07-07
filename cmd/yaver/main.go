// Command yaver is the single API binary. In Phase 0 it serves HTTP and runs
// the flow → mock-provider → outcome path; embedded Hatchet workers, Postgres,
// and the LiveKit adapter are wired in as that infra lands.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebnsina/yaver-api/internal/adapter/memory"
	voicemock "github.com/ebnsina/yaver-api/internal/adapter/voice/mock"
	"github.com/ebnsina/yaver-api/internal/platform/clock"
	"github.com/ebnsina/yaver-api/internal/platform/config"
	"github.com/ebnsina/yaver-api/internal/service/calls"
	httptransport "github.com/ebnsina/yaver-api/internal/transport/http"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}

	// Wire dependencies (ports → adapters). Swap memory/mock for
	// postgres/livekit without touching the service layer.
	provider := voicemock.New(log)
	callRepo := memory.NewCallRepo()
	callsSvc := calls.New(provider, callRepo, clock.Real{})

	handler := httptransport.New(log, callsSvc)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("yaver-api listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}
}
