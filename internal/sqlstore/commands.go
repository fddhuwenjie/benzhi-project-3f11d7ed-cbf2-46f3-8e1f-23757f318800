package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"time"
)

func (s *Store) Create(ctx context.Context, m application.CommandMeta, item *domain.ConservationCase) (result *domain.ConservationCase, replayed bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if old, ok, e := replay(tx, m, ""); e != nil {
		return nil, false, e
	} else if ok {
		_ = tx.Rollback()
		return old, true, nil
	}
	body, err := json.Marshal(item)
	if err != nil {
		return nil, false, err
	}
	if err = domain.ValidateSnapshot(item); err != nil {
		return nil, false, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `INSERT INTO conservation_cases(id,state,revision,snapshot,created_at,updated_at) VALUES(?,?,?,?,?,?)`, item.ID, item.State, item.Revision, body, now, now); err != nil {
		return nil, false, err
	}
	if err = syncProjections(ctx, tx, item); err != nil {
		return nil, false, err
	}
	if err = appendAuditTx(ctx, tx, item, m); err != nil {
		return nil, false, err
	}
	if err = saveReplay(tx, m, item); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return item, false, nil
}

func (s *Store) Mutate(ctx context.Context, id string, m application.CommandMeta, fn func(*domain.ConservationCase) error) (result *domain.ConservationCase, replayed bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if old, ok, e := replay(tx, m, id); e != nil {
		return nil, false, e
	} else if ok {
		_ = tx.Rollback()
		return old, true, nil
	}
	var body []byte
	var revision int64
	if err = tx.QueryRowContext(ctx, `SELECT snapshot,revision FROM conservation_cases WHERE id=?`, id).Scan(&body, &revision); err == sql.ErrNoRows {
		return nil, false, domain.NotFound("个案不存在")
	}
	if err != nil {
		return nil, false, err
	}
	if m.ExpectedRevision != revision {
		return nil, false, application.ErrRevisionConflict
	}
	var item domain.ConservationCase
	if err = json.Unmarshal(body, &item); err != nil {
		return nil, false, err
	}
	if mutationErr := fn(&item); mutationErr != nil {
		rejectedEvent := ""
		switch m.EventType {
		case "case.released":
			rejectedEvent = "release.precheck_rejected"
		case "ethics.approved", "ethics.returned":
			rejectedEvent = "ethics.gate_blocked"
		}
		if rejectedEvent != "" {
			m.EventType = rejectedEvent
			if err = appendAuditWithReasonTx(ctx, tx, &item, m, mutationErr.Error()); err != nil {
				return nil, false, err
			}
			if err = tx.Commit(); err != nil {
				return nil, false, err
			}
		}
		return nil, false, mutationErr
	}
	item.Revision = revision + 1
	if err = domain.ValidateSnapshot(&item); err != nil {
		return nil, false, err
	}
	body, err = json.Marshal(&item)
	if err != nil {
		return nil, false, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE conservation_cases SET state=?,revision=?,snapshot=?,updated_at=? WHERE id=? AND revision=?`, item.State, item.Revision, body, time.Now().UTC().Format(time.RFC3339Nano), id, revision)
	if err != nil {
		return nil, false, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, false, application.ErrRevisionConflict
	}
	if err = syncProjections(ctx, tx, &item); err != nil {
		return nil, false, err
	}
	if err = appendAuditTx(ctx, tx, &item, m); err != nil {
		return nil, false, err
	}
	if err = saveReplay(tx, m, &item); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return &item, false, nil
}
