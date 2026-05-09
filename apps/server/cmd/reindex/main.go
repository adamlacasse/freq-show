package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adamlacasse/freq-show/apps/server/pkg/config"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/discovery"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/embeddings"
)

const ReindexBatchSave = 25

func main() {
	limit := flag.Int("limit", 0, "maximum albums to backfill; 0 means no limit")
	dryRun := flag.Bool("dry-run", false, "print the work plan without making provider calls or DB writes")
	pruneOld := flag.Bool("prune-old", false, "delete embeddings for models other than the current model after backfill")
	flag.Parse()

	if err := run(*limit, *dryRun, *pruneOld); err != nil {
		log.Printf("reindex failed: %v", err)
		os.Exit(1)
	}
}

func run(limit int, dryRun bool, pruneOld bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Database.Driver != "sqlite" {
		return dbErr("reindex requires sqlite database driver")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := db.NewSQLiteStore(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			log.Printf("store close failed: %v", err)
		}
	}()

	embedder, err := embeddings.NewFromConfig(embeddings.Config{
		Provider: cfg.Discovery.EmbeddingsProvider,
		APIKey:   cfg.Discovery.EmbeddingsAPIKey,
		Model:    cfg.Discovery.EmbeddingsModel,
		BaseURL:  cfg.Discovery.EmbeddingsBaseURL,
	})
	if err != nil {
		return err
	}

	albums, err := store.ListAlbumsMissingEmbedding(ctx, embedder.Model(), limit)
	if err != nil {
		return err
	}
	log.Printf("discovery reindex: %d albums missing embeddings for model %s", len(albums), embedder.Model())
	if dryRun {
		if pruneOld {
			log.Printf("discovery reindex: dry run would prune embeddings for models other than %s", embedder.Model())
		}
		return nil
	}

	processed := 0
	skipped := 0
	for _, album := range albums {
		select {
		case <-ctx.Done():
			log.Printf("discovery reindex: interrupted after %d processed, %d skipped", processed, skipped)
			return nil
		default:
		}

		text := discovery.BuildAlbumEmbeddingText(&album, nil)
		if text == "" {
			skipped++
			log.Printf("discovery reindex: skipping thin album %s", album.ID)
			continue
		}
		vec, err := embedder.EncodeBatch(ctx, []string{text})
		if err != nil {
			return err
		}
		if len(vec) != 1 || len(vec[0]) == 0 {
			return dbErr("embedding provider returned no vector")
		}
		if err := store.SaveEmbedding(ctx, album.ID, embedder.Model(), vec[0]); err != nil {
			return err
		}
		processed++
		if processed%ReindexBatchSave == 0 {
			log.Printf("discovery reindex: saved %d embeddings (%d skipped)", processed, skipped)
		}
	}

	if pruneOld {
		deleted, err := store.DeleteOtherModels(ctx, embedder.Model())
		if err != nil {
			return err
		}
		log.Printf("discovery reindex: pruned %d old-model embeddings", deleted)
	}
	log.Printf("discovery reindex: complete; saved %d embeddings, skipped %d thin albums", processed, skipped)
	return nil
}

type dbErr string

func (e dbErr) Error() string {
	return string(e)
}
