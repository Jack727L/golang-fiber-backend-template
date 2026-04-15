package jobs

import (
	"context"
	"time"

	"github.com/yourusername/go-api-starter/core"
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// JobExecutor contains business logic that can be called from both HTTP handlers
// and the background Redis consumer, without duplication.
type JobExecutor struct {
	database *db.DB
}

func NewJobExecutor(database *db.DB) *JobExecutor {
	return &JobExecutor{database: database}
}

// UpdateUserLastActive updates the user's last_active_at timestamp.
// Safe to call synchronously from a handler or asynchronously from a consumer.
func (e *JobExecutor) UpdateUserLastActive(ctx context.Context, userID int32, timestamp time.Time) error {
	_, err := e.database.Queries.UpdateUserLastActive(ctx, db.UpdateUserLastActiveParams{
		ID: userID,
		LastActiveAt: pgtype.Timestamptz{
			Time:  timestamp,
			Valid: true,
		},
	})
	if err != nil {
		core.LogError(nil, err)
	}
	return err
}
