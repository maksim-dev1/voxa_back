// Package store owns storage-relay's Postgres writes (job creation) and
// Redis queue push. It only ever inserts jobs — status updates from there
// on are the worker's job.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const QueueKey = "voxa:jobs:queue"

type Store struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func Connect(ctx context.Context, databaseURL, redisURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Store{pool: pool, redis: rdb}, nil
}

func (s *Store) Close() {
	s.pool.Close()
	_ = s.redis.Close()
}

// CreateJob inserts the job row and pushes its id onto the queue. Both
// happen here (not split across two calls) so a job is never queued
// without a corresponding row the worker can update.
func (s *Store) CreateJob(ctx context.Context, jobID, userID, audioPath string) error {
	_, err := s.pool.Exec(ctx,
		`insert into jobs (id, user_id, status, audio_path) values ($1, $2, 'queued', $3)`,
		jobID, userID, audioPath,
	)
	if err != nil {
		return err
	}

	return s.redis.LPush(ctx, QueueKey, jobID).Err()
}
