package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/adamlacasse/freq-show/apps/server/pkg/api"
	"github.com/adamlacasse/freq-show/apps/server/pkg/config"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/musicbrainz"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/wikipedia"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	baseCtx := context.Background()

	var store db.Store
	switch cfg.Database.Driver {
	case "memory":
		store, err = db.NewMemoryStore(baseCtx)
	case "sqlite":
		store, err = db.NewSQLiteStore(baseCtx, cfg.Database.URL)
	default:
		log.Fatalf("unsupported database driver: %s", cfg.Database.Driver)
	}
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			log.Printf("store close failed: %v", err)
		}
	}()

	mbClient, err := musicbrainz.New(baseCtx, musicbrainz.Config{
		BaseURL:    cfg.MusicBrainz.BaseURL,
		AppName:    cfg.MusicBrainz.AppName,
		AppVersion: cfg.MusicBrainz.AppVersion,
		Contact:    cfg.MusicBrainz.Contact,
		Timeout:    cfg.MusicBrainz.Timeout,
	})
	if err != nil {
		log.Fatalf("musicbrainz client init failed: %v", err)
	}

	wikiClient, err := wikipedia.New(baseCtx, wikipedia.Config{
		BaseURL:   cfg.Wikipedia.BaseURL,
		UserAgent: cfg.Wikipedia.UserAgent,
		Timeout:   cfg.Wikipedia.Timeout,
	})
	if err != nil {
		log.Fatalf("wikipedia client init failed: %v", err)
	}

	router := api.NewRouter(api.RouterConfig{
		MusicBrainz: mbClient,
		Wikipedia:   wikiClient,
		Artists:     store,
		Albums:      store,
	})

	srv := &http.Server{
		Addr:    cfg.Address(),
		Handler: router,
	}

	go func() {
		log.Printf("freqshow backend listening on %s (env=%s)", srv.Addr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("freqshow backend exiting")
}
