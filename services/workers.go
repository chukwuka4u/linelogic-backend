package services

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateQueueDB(name string, department string, userID string, pool *pgxpool.Pool) (string, error) {
	if pool == nil {
		fmt.Println("Error: Received a nil database pool pointer!")
		return "", fmt.Errorf("database pool is nil")
	}
	tx, err := pool.Begin(Ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(Ctx)

	var id string
	err = tx.QueryRow(Ctx,
		`INSERT INTO queues (name, department, user_id) VALUES ($1, $2) RETURNING id`,
		name, department, userID,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	if err = tx.Commit(Ctx); err != nil {
		return "", err
	}

	return id, nil
}

// JoinUpdateDB creates a membership record to track queue membership in Postgres.
func JoinUpdateDB(queueID int, userID string) error {
	tx, err := DB.Begin(Ctx)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback(Ctx)

	_, err = tx.Exec(Ctx,
		`INSERT INTO members (queue_id, user_id) VALUES ($1, $2)`,
		queueID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert member failed: %w", err)
	}

	if err = tx.Commit(Ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

func ValidUser(username string, userId string, pool *pgxpool.Pool) (string, error) {
	var id string
	err := pool.QueryRow(Ctx,
		`SELECT id FROM users WHERE username = $1 AND id = $2`,
		username, userId,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func CreateUser(username string, userId string, phone string) (string, error) {
	var id string
	err := DB.QueryRow(Ctx,
		`INSERT INTO users (username, id, phone) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING RETURNING id`,
		username, userId, phone,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

var schema = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username VARCHAR(20) NOT NULL,
		phone VARCHAR(20) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS queues (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(20) NOT NULL,
		department VARCHAR(20) NOT NULL,
		status BOOLEAN NOT NULL DEFAULT false,
		user_id TEXT NOT NULL,
		CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_queues_user_id ON queues(user_id);
`

func MigrateAction() error {
	_, err := DB.Exec(Ctx,
		schema,
	)
	if err != nil {
		return err
	}
	return nil
}
