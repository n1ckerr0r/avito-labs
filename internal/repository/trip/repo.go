// Package trip implements trip persistence on top of PostgreSQL.
package trip

import (
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

var sql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
