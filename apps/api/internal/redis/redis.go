package redis

import (
	"context"
	"fmt"
	"time"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/logger"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	*goredis.Client
	miniServer *miniredis.Miniredis
	isEmbedded bool
}

type HealthInfo struct {
	Status   string        `json:"status"`
	Latency  time.Duration `json:"latency"`
	Mode     string        `json:"mode"`
	Error    string        `json:"error,omitempty"`
}

func Connect(cfg *config.Config) (*Client, error) {
	log := logger.Get()

	// If no Redis URL specified or explicitly configured as embedded
	if cfg.RedisURL == "" || cfg.RedisURL == "embedded" || cfg.RedisURL == "standalone" {
		log.Info("Starting in-process embedded Redis server for standalone development/testing")
		mini, err := miniredis.Run()
		if err != nil {
			return nil, fmt.Errorf("failed to start embedded miniredis: %w", err)
		}

		rdb := goredis.NewClient(&goredis.Options{
			Addr: mini.Addr(),
		})

		return &Client{
			Client:     rdb,
			miniServer: mini,
			isEmbedded: true,
		}, nil
	}

	opts, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}
	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}

	rdb := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("Could not connect to external Redis; falling back to embedded Redis", "err", err)
		mini, miniErr := miniredis.Run()
		if miniErr != nil {
			return nil, fmt.Errorf("failed to connect to Redis and fallback failed: %w", err)
		}
		fallbackClient := goredis.NewClient(&goredis.Options{
			Addr: mini.Addr(),
		})
		return &Client{
			Client:     fallbackClient,
			miniServer: mini,
			isEmbedded: true,
		}, nil
	}

	log.Info("Connected to Redis successfully", "addr", opts.Addr)
	return &Client{
		Client:     rdb,
		isEmbedded: false,
	}, nil
}

func (c *Client) Health(ctx context.Context) HealthInfo {
	if c == nil || c.Client == nil {
		return HealthInfo{
			Status: "down",
			Mode:   "unavailable",
			Error:  "redis client is nil",
		}
	}

	mode := "standalone-cluster"
	if c.isEmbedded {
		mode = "embedded-miniredis"
	}

	start := time.Now()
	err := c.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return HealthInfo{
			Status:  "down",
			Latency: latency,
			Mode:    mode,
			Error:   err.Error(),
		}
	}

	return HealthInfo{
		Status:  "healthy",
		Latency: latency,
		Mode:    mode,
	}
}

func (c *Client) Close() error {
	if c.miniServer != nil {
		c.miniServer.Close()
	}
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}
