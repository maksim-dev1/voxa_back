// Package db wraps the pgx pool and the handful of queries voxa-api needs.
// Two tables it never writes: jobs.audio_path and job creation are owned by
// storage-relay (it's the one with the file), api only reads jobs/transcripts
// and writes users.
package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type DB struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func (d *DB) CreateUser(ctx context.Context, id, email, passwordHash string) error {
	_, err := d.pool.Exec(ctx,
		`insert into users (id, email, password_hash) values ($1, $2, $3)`,
		id, email, passwordHash,
	)
	return err
}

func (d *DB) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := d.pool.QueryRow(ctx,
		`select id, email, password_hash, created_at from users where email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

type Job struct {
	ID        string
	UserID    string
	Status    string
	AudioPath string
	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JobByID returns a job only if it belongs to userID — callers must never
// be able to peek at another user's job by guessing an id.
func (d *DB) JobByID(ctx context.Context, id, userID string) (Job, error) {
	var j Job
	err := d.pool.QueryRow(ctx,
		`select id, user_id, status, audio_path, error, created_at, updated_at
		 from jobs where id = $1 and user_id = $2`,
		id, userID,
	).Scan(&j.ID, &j.UserID, &j.Status, &j.AudioPath, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}

func (d *DB) ListJobs(ctx context.Context, userID string) ([]Job, error) {
	rows, err := d.pool.Query(ctx,
		`select id, user_id, status, audio_path, error, created_at, updated_at
		 from jobs where user_id = $1 order by created_at desc`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.UserID, &j.Status, &j.AudioPath, &j.Error, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

type Transcript struct {
	JobID    string
	Segments []byte // raw jsonb, decoded by the caller/client
}

func (d *DB) TranscriptByJobID(ctx context.Context, jobID string) (Transcript, error) {
	var t Transcript
	err := d.pool.QueryRow(ctx,
		`select job_id, segments from transcripts where job_id = $1`,
		jobID,
	).Scan(&t.JobID, &t.Segments)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}
