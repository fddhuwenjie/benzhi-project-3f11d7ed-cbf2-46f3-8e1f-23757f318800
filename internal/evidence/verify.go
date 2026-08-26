package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"time"
)

func (m *Manager) Verify(ctx context.Context, item *domain.ConservationCase, current []domain.AuditEvent) (application.VerificationResult, error) {
	result := application.VerificationResult{Valid: true, CheckedAt: m.now(), Problems: []string{}}
	raw, err := m.Read(ctx, item.ID)
	if err != nil {
		return result, err
	}
	var b bundle
	if err = json.Unmarshal(raw, &b); err != nil {
		return result, fmt.Errorf("证据包 JSON 无效: %w", err)
	}
	result.ManifestDigest = b.ManifestDigest
	result.AuditChainHead = b.AuditChainHead
	result.EventCount = b.EventCount
	stored := b.ManifestDigest
	b.ManifestDigest = ""
	unsigned, _ := canonical(b)
	if digest(unsigned) != stored {
		result.Problems = append(result.Problems, "manifest_digest 不匹配")
	}
	if item.Archive == nil {
		result.Problems = append(result.Problems, "个案缺少归档清单")
	} else {
		if stored != item.Archive.ManifestDigest {
			result.Problems = append(result.Problems, "数据库与文件的清单摘要不一致")
		}
		if b.CaseRevision != item.Archive.CaseRevision {
			result.Problems = append(result.Problems, "归档 revision 不一致")
		}
	}
	for _, d := range b.Documents {
		if digest(d.Content) != d.SHA256 {
			result.Problems = append(result.Problems, "文档摘要变化: "+d.Name)
		}
	}
	if int64(len(b.Events)) != b.EventCount {
		result.Problems = append(result.Problems, "事件数量不一致")
	}
	previous := ""
	for i, e := range b.Events {
		if e.Sequence != int64(i+1) || e.PreviousHash != previous || domain.HashEvent(previous, e.Sequence, e.Revision, e.CaseID, e.Type, e.ActorID, e.Payload) != e.Hash {
			result.Problems = append(result.Problems, fmt.Sprintf("审计事件 %d 顺序或摘要异常", i+1))
			break
		}
		previous = e.Hash
	}
	if previous != b.AuditChainHead {
		result.Problems = append(result.Problems, "审计链头不一致")
	}
	if len(current) < len(b.Events) {
		result.Problems = append(result.Problems, "数据库审计事件缺失")
	} else {
		for i, e := range b.Events {
			if current[i].Hash != e.Hash {
				result.Problems = append(result.Problems, "数据库与证据包审计事件不一致")
				break
			}
		}
	}
	result.Valid = len(result.Problems) == 0
	result.CheckedAt = time.Now().UTC()
	return result, nil
}
