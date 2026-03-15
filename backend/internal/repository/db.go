package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxTx is satisfied by both *pgxpool.Pool and pgx.Tx so repository methods
// can be called with or without a transaction.
type pgxTx interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// queryRow dispatches to the transaction if provided, otherwise to the pool.
func queryRow(db *pgxpool.Pool, tx pgxTx, sql string, args ...any) pgx.Row {
	if tx != nil {
		return tx.QueryRow(context.Background(), sql, args...)
	}
	return db.QueryRow(context.Background(), sql, args...)
}

// execQuery dispatches Exec to tx or pool.
func execQuery(db *pgxpool.Pool, tx pgxTx, sql string, args ...any) (int64, error) {
	var ct pgconn.CommandTag
	var err error
	if tx != nil {
		ct, err = tx.Exec(context.Background(), sql, args...)
	} else {
		ct, err = db.Exec(context.Background(), sql, args...)
	}
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
