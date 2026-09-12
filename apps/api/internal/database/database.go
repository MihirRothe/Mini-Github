package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	*sql.DB
	isStandalone bool
}

type HealthInfo struct {
	Status      string        `json:"status"`
	Latency     time.Duration `json:"latency"`
	OpenConns   int           `json:"open_conns"`
	InUseConns  int           `json:"in_use_conns"`
	IdleConns   int           `json:"idle_conns"`
	Driver      string        `json:"driver"`
	Error       string        `json:"error,omitempty"`
}

func Connect(cfg *config.Config) (*DB, error) {
	log := logger.Get()

	if cfg.DatabaseURL == "" || cfg.DatabaseURL == "embedded" || cfg.DatabaseURL == "standalone" {
		log.Warn("No PostgreSQL DATABASE_URL configured - running database in standalone mode")
		return &DB{DB: nil, isStandalone: true}, nil
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMins) * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Warn("Failed to connect to PostgreSQL; fallback to standalone mode", "err", err)
		return &DB{DB: db, isStandalone: false}, nil
	}

	log.Info("Successfully connected to PostgreSQL database")
	return &DB{DB: db, isStandalone: false}, nil
}

func (d *DB) IsStandalone() bool {
	return d == nil || d.isStandalone || d.DB == nil
}

func (d *DB) Health(ctx context.Context) HealthInfo {
	if d.IsStandalone() {
		return HealthInfo{
			Status: "standalone",
			Driver: "in-memory-fallback",
		}
	}

	start := time.Now()
	err := d.PingContext(ctx)
	latency := time.Since(start)

	stats := d.Stats()
	if err != nil {
		return HealthInfo{
			Status:     "down",
			Latency:    latency,
			OpenConns:  stats.OpenConnections,
			InUseConns: stats.InUse,
			IdleConns:  stats.Idle,
			Driver:     "postgres",
			Error:      err.Error(),
		}
	}

	return HealthInfo{
		Status:     "healthy",
		Latency:    latency,
		OpenConns:  stats.OpenConnections,
		InUseConns: stats.InUse,
		IdleConns:  stats.Idle,
		Driver:     "postgres",
	}
}
