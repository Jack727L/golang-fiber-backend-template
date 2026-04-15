package jobs

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	"github.com/yourusername/go-api-starter/core"
	"github.com/yourusername/go-api-starter/env"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// RedisClient is the shared Redis client. Nil in sync-test mode.
var RedisClient *redis.Client

// InitRedis initialises the Redis client. Skipped in sync test mode.
func InitRedis() {
	if env.IsTestMode() && !env.IsAsyncMode() {
		core.LogDebug(nil, "Test mode: skipping Redis initialisation (use ASYNC_MODE=true to enable)")
		return
	}

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")
	tlsEnabled := os.Getenv("REDIS_TLS_ENABLED") == "true"

	if host == "" || port == "" {
		log.Fatal("Missing required Redis environment variables: REDIS_HOST, REDIS_PORT")
	}

	opts := &redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       0,
	}

	if tlsEnabled {
		opts.TLSConfig = &tls.Config{ServerName: host}
	}

	RedisClient = redis.NewClient(opts)

	if _, err := RedisClient.Ping(ctx).Result(); err != nil && !env.IsLocalDevMode() {
		log.Fatalf("Redis connection failed: %v", err)
	}
}

// ─── Example job: last-active update ─────────────────────────────────────────

// EnqueueLastActive enqueues a background job to update a user's last_active_at.
func EnqueueLastActive(userID int32) error {
	if env.IsTestMode() && !env.IsAsyncMode() {
		return nil
	}

	cmd := RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "last_active_stream",
		Values: map[string]interface{}{
			"user_id":        userID,
			"last_active_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	return cmd.Err()
}

// ─── Generic job helper ───────────────────────────────────────────────────────

// EnqueueJob sends an arbitrary job payload to the named stream.
// Use this as a building block when adding new job types.
func EnqueueJob(stream string, values map[string]interface{}) error {
	if env.IsTestMode() && !env.IsAsyncMode() {
		return nil
	}
	return RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}
