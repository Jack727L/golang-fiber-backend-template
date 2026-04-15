package jobs

import (
	"context"
	"time"

	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/env"
)

// SmartExecutor transparently switches between synchronous (test mode) and
// asynchronous (production) execution. Add a method here for every new job type.
type SmartExecutor struct {
	executor *JobExecutor
}

func NewSmartExecutor(database *db.DB) *SmartExecutor {
	return &SmartExecutor{executor: NewJobExecutor(database)}
}

// UpdateUserLastActive runs the update immediately in test mode,
// enqueues it to Redis in production (with sync fallback on Redis failure).
func (s *SmartExecutor) UpdateUserLastActive(ctx context.Context, userID int32) error {
	if env.IsTestMode() && !env.IsAsyncMode() {
		return s.executor.UpdateUserLastActive(ctx, userID, time.Now())
	}

	if err := EnqueueLastActive(userID); err != nil {
		// Redis unavailable — fall back to synchronous execution
		return s.executor.UpdateUserLastActive(ctx, userID, time.Now())
	}
	return nil
}

// ForceSync bypasses the async path and runs immediately.
// Useful for critical operations where you cannot wait for the queue.
func (s *SmartExecutor) ForceSync() *JobExecutor {
	return s.executor
}
