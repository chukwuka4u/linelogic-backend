package services

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateQueueDB(name string, department string, pool *pgxpool.Pool) (string, error) {
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
		`INSERT INTO queues (name, department) VALUES ($1, $2) RETURNING id`,
		name, department,
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
