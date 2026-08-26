package sqlstore

import (
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
)

func replay(tx *sql.Tx, m application.CommandMeta) (*domain.ConservationCase, bool, error) {
	var fp string
	var body []byte
	err := tx.QueryRow(`SELECT fingerprint,response FROM idempotency WHERE request_id=?`, m.RequestID).Scan(&fp, &body)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if fp != m.Fingerprint {
		return nil, false, application.ErrIdempotencyConflict
	}
	var item domain.ConservationCase
	if err = json.Unmarshal(body, &item); err != nil {
		return nil, false, err
	}
	return &item, true, nil
}
func saveReplay(tx *sql.Tx, m application.CommandMeta, item *domain.ConservationCase) error {
	body, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO idempotency(request_id,fingerprint,case_id,response,created_at) VALUES(?,?,?,?,CURRENT_TIMESTAMP)`, m.RequestID, m.Fingerprint, item.ID, body)
	return err
}
