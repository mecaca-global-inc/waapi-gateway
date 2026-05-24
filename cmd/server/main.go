package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	wastore "go.mau.fi/whatsmeow/store"

	"github.com/mecaca/waapi-gateway/internal/api"
	"github.com/mecaca/waapi-gateway/internal/config"
	mcpserver "github.com/mecaca/waapi-gateway/internal/mcp"
	"github.com/mecaca/waapi-gateway/internal/observability"
	"github.com/mecaca/waapi-gateway/internal/store"
	"github.com/mecaca/waapi-gateway/internal/wa"
	"github.com/mecaca/waapi-gateway/internal/webhook"
)

// Linked-device label shown in the WhatsApp app under "Linked Devices".
// Override via env DEVICE_NAME if you fork/rebrand.
var deviceName = func() string {
	if v := os.Getenv("DEVICE_NAME"); v != "" {
		return v
	}
	return "WAAPI Gateway"
}()

func main() {
	// Subcommand dispatch (kept stdlib-simple — only one subcommand today).
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err := mcpserver.Run(os.Args[2:]); err != nil {
			log.Fatal().Err(err).Msg("mcp server")
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	observability.Setup(cfg.LogLevel, cfg.LogFormat)
	// Identify this client to WhatsApp so the Linked Devices screen shows
	// "WAAPI Gateway" instead of the whatsmeow default.
	wastore.SetOSInfo(deviceName, [3]uint32{1, 0, 0})
	log.Info().Str("addr", cfg.HTTPAddr).Str("db", cfg.DBDialect).Str("device", deviceName).Msg("starting waapi-gateway")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := store.Open(ctx, cfg.DBDialect, cfg.DBURI)
	if err != nil {
		log.Fatal().Err(err).Msg("open db")
	}
	defer db.Close()
	if err := store.Migrate(db, cfg.DBDialect); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}
	repo := store.NewRepo(db)

	disp := webhook.New(repo)
	disp.Start(ctx, 4)

	mgr, err := wa.NewManager(ctx, cfg.DBDialect, cfg.DBURI, repo, disp)
	if err != nil {
		log.Fatal().Err(err).Msg("wa manager")
	}
	if err := mgr.LoadExisting(ctx); err != nil {
		log.Error().Err(err).Msg("load existing sessions")
	}
	// Auto-start any session that has saved credentials.
	for _, sess := range mgr.List() {
		if sess.Client.Store.ID != nil {
			if err := sess.Start(context.Background()); err != nil {
				log.Error().Err(err).Str("session", sess.Name).Msg("auto-start failed")
			}
		}
	}

	// Honour PaaS-injected PORT envs (Heroku, Render, Fly, Railway, xCloud, etc.).
	addr := cfg.HTTPAddr
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Info().Str("listen", addr).Msg("http listen")

	srv := api.NewServer(cfg, repo, mgr)
	go func() {
		if err := srv.Listen(addr); err != nil {
			log.Error().Err(err).Msg("http server stopped")
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = srv.Shutdown()
	mgr.Close()
	_ = shutCtx
	log.Info().Msg("bye")
}
