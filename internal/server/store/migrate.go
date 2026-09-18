package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/mtiluk/potok/internal/server/auth"
	"github.com/mtiluk/potok/migrations"
)

func (s *Store) migrator() (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("store: read migrations: %w", err)
	}

	driver, err := sqlite.WithInstance(s.db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("store: migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("store: migrator: %w", err)
	}
	return m, nil
}

func (s *Store) Migrate() error {
	m, err := s.migrator()
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("store: apply migrations: %w", err)
	}
	if err := s.hashPlaintextAPIKeys(context.Background()); err != nil {
		return err
	}
	return nil
}

func (s *Store) hashPlaintextAPIKeys(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, api_key_hash
		FROM users
		WHERE api_key_hash IS NOT NULL AND api_key_hash != ''`)
	if err != nil {
		return fmt.Errorf("store: hash plaintext api keys: %w", err)
	}
	defer rows.Close()

	type pending struct {
		id, stored string
	}
	var toHash []pending
	for rows.Next() {
		var row pending
		if err := rows.Scan(&row.id, &row.stored); err != nil {
			return fmt.Errorf("store: hash plaintext api keys: %w", err)
		}
		if !auth.IsHashedAPIKey(row.stored) {
			toHash = append(toHash, row)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: hash plaintext api keys: %w", err)
	}

	for _, row := range toHash {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE users SET api_key_hash = ? WHERE id = ?`,
			auth.HashAPIKey(row.stored), row.id,
		); err != nil {
			return fmt.Errorf("store: hash plaintext api keys: %w", err)
		}
	}
	return nil
}

func (s *Store) Version() (version uint, dirty bool, err error) {
	m, err := s.migrator()
	if err != nil {
		return 0, false, err
	}

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("store: read schema version: %w", err)
	}
	return version, dirty, nil
}
