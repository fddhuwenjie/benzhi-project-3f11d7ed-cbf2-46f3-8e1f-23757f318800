package application

import "errors"

var (
	ErrRevisionConflict    = errors.New("expected_revision 与当前 revision 不一致")
	ErrIdempotencyConflict = errors.New("request_id 已用于不同请求")
	ErrAuditCorrupt        = errors.New("审计哈希链不连续")
	ErrProjectionCorrupt   = errors.New("规范化业务投影与聚合快照不一致")
)
