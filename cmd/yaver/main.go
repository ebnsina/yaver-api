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

	chatbuiltin "github.com/ebnsina/yaver-api/internal/adapter/chat/builtin"
	reporterbuiltin "github.com/ebnsina/yaver-api/internal/adapter/reporter/builtin"
	logsender "github.com/ebnsina/yaver-api/internal/adapter/messaging/logsender"
	metasender "github.com/ebnsina/yaver-api/internal/adapter/messaging/meta"
	hatchetorch "github.com/ebnsina/yaver-api/internal/adapter/orchestrator/hatchet"
	orchlocal "github.com/ebnsina/yaver-api/internal/adapter/orchestrator/local"
	"github.com/ebnsina/yaver-api/internal/adapter/postgres"
	voicemock "github.com/ebnsina/yaver-api/internal/adapter/voice/mock"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/platform/clock"
	"github.com/ebnsina/yaver-api/internal/platform/config"
	"github.com/ebnsina/yaver-api/internal/platform/db"
	"github.com/ebnsina/yaver-api/internal/service/analytics"
	"github.com/ebnsina/yaver-api/internal/service/apikeys"
	"github.com/ebnsina/yaver-api/internal/service/auth"
	"github.com/ebnsina/yaver-api/internal/service/billing"
	"github.com/ebnsina/yaver-api/internal/service/calls"
	"github.com/ebnsina/yaver-api/internal/service/campaigns"
	"github.com/ebnsina/yaver-api/internal/service/chat"
	"github.com/ebnsina/yaver-api/internal/service/customers"
	"github.com/ebnsina/yaver-api/internal/service/flows"
	"github.com/ebnsina/yaver-api/internal/service/ingest"
	"github.com/ebnsina/yaver-api/internal/service/messaging"
	"github.com/ebnsina/yaver-api/internal/service/reports"
	"github.com/ebnsina/yaver-api/internal/service/webhooks"
	httptransport "github.com/ebnsina/yaver-api/internal/transport/http"
	"github.com/ebnsina/yaver-api/pkg/crypto"
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
	cipher, err := crypto.New(cfg.EncryptionKey)
	if err != nil {
		log.Error("encryption key", "err", err)
		os.Exit(1)
	}

	flowRepo := postgres.NewFlowRepo(pool)
	authSvc := auth.New(postgres.NewAuthRepo(pool), clock.Real{}, cfg.AuthSecret, cfg.Env)
	creditRepo := postgres.NewCreditRepo(pool)
	callRepo := postgres.NewCallRepo(pool)
	callPolicyRepo := postgres.NewCallPolicyRepo(pool)
	callsSvc := calls.New(voicemock.New(log), postgres.NewOutcomeRepo(pool), callRepo, flowRepo, creditRepo, callPolicyRepo, clock.Real{})
	billingSvc := billing.New(creditRepo)
	analyticsSvc := analytics.New(callRepo, creditRepo, postgres.NewAnalyticsRepo(pool))
	reportsSvc := reports.New(analyticsSvc, reporterbuiltin.New())
	flowsSvc := flows.New(flowRepo)

	// Orchestrator: Hatchet (durable, fairness-keyed) or the in-process local
	// dispatcher. Both satisfy domain.Orchestrator.
	var orch domain.Orchestrator
	switch cfg.Orchestrator {
	case "hatchet":
		ho, err := hatchetorch.New(callsSvc.PlaceCall)
		if err != nil {
			log.Error("hatchet client", "err", err)
			os.Exit(1)
		}
		go func() {
			if err := ho.StartWorker(context.Background()); err != nil {
				log.Error("hatchet worker stopped", "err", err)
			}
		}()
		orch = ho
		log.Info("orchestrator: hatchet")
	default:
		lo := orchlocal.New(log, 8, callsSvc.PlaceCall)
		defer lo.Shutdown()
		orch = lo
		log.Info("orchestrator: local")
	}

	keysSvc := apikeys.New(postgres.NewAPIKeyRepo(pool))
	custRepo := postgres.NewCustomerRepo(pool)
	custSvc := customers.New(custRepo)
	campSvc := campaigns.New(postgres.NewCampaignRepo(pool), custRepo, flowRepo, orch)
	ingestSvc := ingest.New(postgres.NewEventRepo(pool), custRepo, flowRepo, orch)

	// Chat channel — provider-agnostic model behind domain.ChatModel.
	var chatModel domain.ChatModel
	switch cfg.ChatProvider {
	case "builtin":
		chatModel = chatbuiltin.New()
	default:
		log.Error("unsupported YAVER_CHAT_PROVIDER (only 'builtin' is implemented)", "value", cfg.ChatProvider)
		os.Exit(1)
	}
	chatSvc := chat.New(postgres.NewChatRepo(pool), postgres.NewChatSettingsRepo(pool), chatModel, chatbuiltin.NewSummarizer(), postgres.NewInsightRepo(pool))

	// Messaging channels — pluggable sender behind domain.MessagingSender.
	var msgSender domain.MessagingSender
	switch cfg.MsgSender {
	case "log":
		msgSender = logsender.New(log)
	case "meta":
		msgSender = metasender.New()
	default:
		log.Error("unsupported YAVER_MSG_SENDER (want 'log' or 'meta')", "value", cfg.MsgSender)
		os.Exit(1)
	}
	msgSvc := messaging.New(postgres.NewChannelRepo(pool), chatSvc, msgSender, cipher, log)

	// Webhook dispatcher: drains the outbox and delivers, on its own loop.
	webhooksSvc := webhooks.New(postgres.NewWebhookRepo(pool), cipher, log)
	go webhooksSvc.Run(context.Background())

	orgProv := postgres.NewOrgRepo(pool)
	handler := httptransport.New(log, cfg.Env, authSvc, orgProv, callsSvc, flowsSvc, custSvc, campSvc, chatSvc, msgSvc, billingSvc, analyticsSvc, reportsSvc, keysSvc, ingestSvc, webhooksSvc, orch)
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
