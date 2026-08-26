package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"time"
)

func appendAuditTx(ctx context.Context, tx *sql.Tx, item *domain.ConservationCase, m application.CommandMeta) error {
	return appendAuditWithReasonTx(ctx, tx, item, m, "")
}

func appendAuditWithReasonTx(ctx context.Context, tx *sql.Tx, item *domain.ConservationCase, m application.CommandMeta, reason string) error {
	var seq int64
	var previous string
	err := tx.QueryRowContext(ctx, `SELECT sequence,event_hash FROM audit_events WHERE case_id=? ORDER BY sequence DESC LIMIT 1`, item.ID).Scan(&seq, &previous)
	if err == sql.ErrNoRows {
		seq = 0
		previous = ""
	} else if err != nil {
		return err
	}
	seq++
	payload, _ := json.Marshal(struct {
		State     domain.CaseState `json:"state"`
		Revision  int64            `json:"revision"`
		RequestID string           `json:"request_id"`
		Reason    string           `json:"reason,omitempty"`
	}{item.State, item.Revision, m.RequestID, reason})
	hash := domain.HashEvent(previous, seq, item.Revision, item.ID, m.EventType, m.ActorID, payload)
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(case_id,sequence,revision,event_type,actor_id,occurred_at,payload,previous_hash,event_hash) VALUES(?,?,?,?,?,?,?,?,?)`, item.ID, seq, item.Revision, m.EventType, m.ActorID, time.Now().UTC().Format(time.RFC3339Nano), payload, previous, hash)
	return err
}

func (s *Store) Events(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	events, err := s.loadEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.VerifyAudit(ctx, id); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) loadEvents(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sequence,revision,event_type,actor_id,occurred_at,payload,previous_hash,event_hash FROM audit_events WHERE case_id=? ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		var at string
		e.CaseID = id
		if err := rows.Scan(&e.Sequence, &e.Revision, &e.Type, &e.ActorID, &at, &e.Payload, &e.PreviousHash, &e.Hash); err != nil {
			return nil, err
		}
		e.OccurredAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) VerifyAudit(ctx context.Context, id string) (string, int64, error) {
	events, err := s.loadEvents(ctx, id)
	if err != nil {
		return "", 0, err
	}
	head, count, err := verifyEvents(events)
	if err != nil {
		return "", 0, err
	}
	var revision int64
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM conservation_cases WHERE id=?`, id).Scan(&revision); err != nil {
		return "", 0, err
	}
	if count == 0 || events[len(events)-1].Revision != revision {
		return "", 0, application.ErrAuditCorrupt
	}
	return head, count, nil
}

func verifyEvents(events []domain.AuditEvent) (string, int64, error) {
	previous := ""
	var revision int64
	for i, e := range events {
		if e.Sequence != int64(i+1) || e.Revision < revision || e.Revision > revision+1 || e.PreviousHash != previous {
			return "", 0, application.ErrAuditCorrupt
		}
		revision = e.Revision
		expected := domain.HashEvent(previous, e.Sequence, e.Revision, e.CaseID, e.Type, e.ActorID, e.Payload)
		if expected != e.Hash {
			return "", 0, application.ErrAuditCorrupt
		}
		previous = e.Hash
	}
	return previous, int64(len(events)), nil
}
