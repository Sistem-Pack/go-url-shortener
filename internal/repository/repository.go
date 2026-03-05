package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrConflict = errors.New("url conflict")

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

func RunMigrations(db *sql.DB) error {
	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"pgx", driver)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	return nil
}

func (p *PostgresStorage) SaveURL(ctx context.Context, id string, originalURL string) error {
	query := `INSERT INTO urls (id, original_url) VALUES ($1, $2) ON CONFLICT (original_url) DO NOTHING`

	result, err := p.DB.ExecContext(ctx, query, id, originalURL)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrConflict
	}

	return err
}

func (p *PostgresStorage) GetIDByPath(ctx context.Context, originalURL string) (string, error) {
	var id string
	query := `SELECT id FROM urls WHERE original_url = $1`
	err := p.DB.QueryRowContext(ctx, query, originalURL).Scan(&id)
	return id, err
}

func (p *PostgresStorage) GetURL(ctx context.Context, id string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM urls WHERE id = $1`
	err := p.DB.QueryRowContext(ctx, query, id).Scan(&originalURL)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (p *PostgresStorage) SaveBatch(ctx context.Context, data map[string]string) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (id, original_url) VALUES ($1, $2) ON CONFLICT (original_url) DO NOTHING")
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for id, originalURL := range data {
		if _, err := stmt.ExecContext(ctx, id, originalURL); err != nil {
			return fmt.Errorf("exec stmt: %w", err)
		}
	}

	return tx.Commit()
}
