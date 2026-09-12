package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/logger"
	"forgehub/apps/api/internal/redis"
	"forgehub/apps/api/internal/router"
)

func main() {
	cfg := config.Load()
	log := logger.Init(cfg.LogLevel, cfg.LogFormat)

	log.Info("==================================================")
	log.Info("          Starting ForgeHub API Server            ")
	log.Info("==================================================",
		"env", cfg.Env,
		"port", cfg.Port,
		"addr", cfg.Addr(),
		"git_root", cfg.GitRootDir,
	)

	// Initialize Database Connection
	db, err := database.Connect(cfg)
	if err != nil {
		log.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}

	// Locate and Execute Migrations
	migrationsPath := findMigrationsDir()
	log.Info("Scanning for database migrations", "path", migrationsPath)
	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := database.RunMigrations(migrationCtx, db, migrationsPath); err != nil {
		migrationCancel()
		log.Error("Failed to run database migrations", "err", err)
		os.Exit(1)
	}
	migrationCancel()

	// Initialize Redis Connection
	rdb, err := redis.Connect(cfg)
	if err != nil {
		log.Error("Failed to initialize Redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Build HTTP Router
	httpRouter := router.New(router.RouterOptions{
		Config: cfg,
		DB:     db,
		Redis:  rdb,
	})

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      httpRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server runner channel
	serverErrors := make(chan error, 1)
	go func() {
		log.Info(fmt.Sprintf("ForgeHub API listening on http://%s", cfg.Addr()))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Graceful Shutdown on SIGINT or SIGTERM
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("Critical error starting server", "err", err)
		os.Exit(1)

	case sig := <-shutdown:
		log.Info("Received termination signal, initiating graceful shutdown", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Server forced to shutdown after timeout", "err", err)
			_ = srv.Close()
		}

		if db.DB != nil {
			_ = db.Close()
		}

		log.Info("ForgeHub API Server gracefully terminated")
	}
}

func findMigrationsDir() string {
	candidates := []string{
		"./migrations",
		"../migrations",
		"../../migrations",
		"/app/migrations",
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				return abs
			}
		}
	}
	return "./migrations"
}
