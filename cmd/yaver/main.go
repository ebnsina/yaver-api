// Command yaver is the single API binary. In Phase 0 it serves HTTP, runs
// phone-OTP auth (Postgres), and the flow → mock-provider → outcome path via an
// in-process orchestrator. The Hatchet adapter and the LiveKit voice adapter
// slot in behind their interfaces as that infra lands.
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
	orchlocal "github.com/ebnsina/yaver-api/internal/adapter/orchestrator/local"
	"github.com/ebnsina/yaver-api/internal/adapter/postgres"
	voicemock "github.com/ebnsina/yaver-api/internal/adapter/voice/mock"
	"github.com/ebnsina/yaver-api/internal/platform/clock"
	"github.com/ebnsina/yaver-api/internal/platform/config"
	"github.com/ebnsina/yaver-api/internal/platform/db"
	"github.com/ebnsina/yaver-api/internal/service/auth"
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

	// DB pool (lazy: dials on first query, so boot doesn't require a live DB).
	pool, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Wire ports → adapters. Swap memory/mock/local for postgres/livekit/hatchet
	// without touching the service layer.
	authSvc := auth.New(postgres.NewAuthRepo(pool), clock.Real{}, cfg.AuthSecret, cfg.Env)
	callsSvc := calls.New(voicemock.New(log), memory.NewCallRepo(), clock.Real{})
	orch := orchlocal.New(log, 8, callsSvc.PlaceCall)
	defer orch.Shutdown()

	handler := httptransport.New(log, cfg.Env, authSvc, callsSvc, orch)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

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
