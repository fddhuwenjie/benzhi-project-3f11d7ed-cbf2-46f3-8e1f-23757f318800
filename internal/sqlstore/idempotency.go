package sqlstore

import (
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
)

// replay returns the previously persisted aggregate for an idempotent request.
// The target case_id participates in the idempotency judgement for mutations:
// a replay is only served when the stored response belongs to the same target
// case. When the same request_id and fingerprint are replayed against a
// different case, ErrIdempotencyConflict is returned so that no foreign
// response is leaked and the target case remains unchanged. For Create, the
// case id is generated server-side and is not part of the request, so an empty
// caseID is passed and only the fingerprint is compared.
func replay(tx *sql.Tx, m application.CommandMeta, caseID string) (*domain.ConservationCase, bool, error) {
	var fp string
	var storedCaseID string
	var body []byte
	err := tx.QueryRow(`SELECT fingerprint,case_id,response FROM idempotency WHERE request_id=?`, m.RequestID).Scan(&fp, &storedCaseID, &body)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if fp != m.Fingerprint || (caseID != "" && storedCaseID != caseID) {
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
