package repository

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	DB *sql.DB
}

func (p *PostgresStorage) Ping() error {
	return p.DB.Ping()
}

func OpenDatabase(dbConnectionString string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dbConnectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
