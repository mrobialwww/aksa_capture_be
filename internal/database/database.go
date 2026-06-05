package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDatabase(url string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), url)
}
