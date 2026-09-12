package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"forgehub/apps/api/internal/logger"
)

type Migration struct {
	Version   string
	Name      string
	SQL       string
	AppliedAt time.Time
}

func RunMigrations(ctx context.Context, db *DB, migrationsDir string) error {
	log := logger.Get()

	if db.IsStandalone() {
		log.Info("Running in standalone mode: skipping database migrations")
		return nil
	}

	// 1. Create schema_migrations table if not exists
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Query already applied migrations
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var ver string
		if err := rows.Scan(&ver); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[ver] = true
	}

	// 3. Scan migrations directory
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Warn("Migrations directory not accessible", "path", migrationsDir, "err", err)
		return nil
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	// 4. Apply pending migrations
	for _, filename := range upFiles {
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue
		}
		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".up.sql")

		if applied[version] {
			continue
		}

		fullPath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		log.Info("Applying database migration", "version", version, "name", name)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin tx for migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", version, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		log.Info("Successfully applied migration", "version", version, "name", name)
	}

	return nil
}
