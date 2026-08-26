package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"manuscript-conservation-gate/internal/application"
	_ "modernc.org/sqlite"
	"time"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	s := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err = s.verifyAll(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL`, `PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(1,CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS conservation_cases(id TEXT PRIMARY KEY,state TEXT NOT NULL,revision INTEGER NOT NULL,snapshot BLOB NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS idempotency(request_id TEXT PRIMARY KEY,fingerprint TEXT NOT NULL,case_id TEXT NOT NULL,response BLOB NOT NULL,created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS audit_events(case_id TEXT NOT NULL,sequence INTEGER NOT NULL,revision INTEGER NOT NULL,event_type TEXT NOT NULL,actor_id TEXT NOT NULL,occurred_at TEXT NOT NULL,payload BLOB NOT NULL,previous_hash TEXT NOT NULL,event_hash TEXT NOT NULL,PRIMARY KEY(case_id,sequence),FOREIGN KEY(case_id) REFERENCES conservation_cases(id))`,
		`CREATE INDEX IF NOT EXISTS idx_cases_state ON conservation_cases(state)`, `CREATE INDEX IF NOT EXISTS idx_audit_revision ON audit_events(case_id,revision)`,
	}
	stmts = append(stmts, projectionSchema()...)
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("初始化 SQLite: %w", err)
		}
	}
	if err := s.ensureProjectionColumns(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureProjectionColumns(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(treatment_plans)`)
	if err != nil {
		return err
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, column := range []struct{ name, ddl string }{{"submitted_at", `ALTER TABLE treatment_plans ADD COLUMN submitted_at TEXT`}, {"review_decision", `ALTER TABLE treatment_plans ADD COLUMN review_decision TEXT NOT NULL DEFAULT ''`}} {
		if !columns[column.name] {
			if _, err := s.db.ExecContext(ctx, column.ddl); err != nil {
				return fmt.Errorf("升级 SQLite 投影: %w", err)
			}
		}
	}
	return nil
}

func (s *Store) verifyAll(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM conservation_cases ORDER BY id`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, _, err := s.VerifyAudit(ctx, id); err != nil {
			return fmt.Errorf("个案 %s: %w", id, err)
		}
		item, err := s.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("个案 %s: %w", id, err)
		}
		if err := s.verifyProjections(ctx, item); err != nil {
			return fmt.Errorf("个案 %s: %w", id, err)
		}
	}
	return nil
}

var _ application.Store = (*Store)(nil)
