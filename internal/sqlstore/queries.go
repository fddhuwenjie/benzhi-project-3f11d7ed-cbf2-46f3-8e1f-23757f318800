package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/domain"
)

func (s *Store) Get(ctx context.Context, id string) (*domain.ConservationCase, error) {
	var body []byte
	if err := s.db.QueryRowContext(ctx, `SELECT snapshot FROM conservation_cases WHERE id=?`, id).Scan(&body); err == sql.ErrNoRows {
		return nil, domain.NotFound("个案不存在")
	} else if err != nil {
		return nil, err
	}
	var item domain.ConservationCase
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, err
	}
	if err := domain.ValidateSnapshot(&item); err != nil {
		return nil, err
	}
	if err := s.verifyProjections(ctx, &item); err != nil {
		return nil, err
	}
	return &item, nil
}
func (s *Store) List(ctx context.Context) ([]domain.ConservationCase, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot FROM conservation_cases ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	bodies := [][]byte{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			rows.Close()
			return nil, err
		}
		bodies = append(bodies, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	out := []domain.ConservationCase{}
	for _, b := range bodies {
		var item domain.ConservationCase
		if err := json.Unmarshal(b, &item); err != nil {
			return nil, err
		}
		if err := domain.ValidateSnapshot(&item); err != nil {
			return nil, err
		}
		if err := s.verifyProjections(ctx, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
