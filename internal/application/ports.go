package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
	"time"
)

type CommandMeta struct {
	RequestID        string
	ExpectedRevision int64
	ActorID          string
	Role             string
	Fingerprint      string
	EventType        string
}

type Store interface {
	Create(context.Context, CommandMeta, *domain.ConservationCase) (*domain.ConservationCase, bool, error)
	Mutate(context.Context, string, CommandMeta, func(*domain.ConservationCase) error) (*domain.ConservationCase, bool, error)
	Get(context.Context, string) (*domain.ConservationCase, error)
	List(context.Context) ([]domain.ConservationCase, error)
	Events(context.Context, string) ([]domain.AuditEvent, error)
	VerifyAudit(context.Context, string) (string, int64, error)
	Close() error
}

type EvidencePort interface {
	Generate(context.Context, *domain.ConservationCase, []domain.AuditEvent, string) (domain.ArchiveManifest, error)
	Read(context.Context, string) ([]byte, error)
	Verify(context.Context, *domain.ConservationCase, []domain.AuditEvent) (VerificationResult, error)
}

type VerificationResult struct {
	Valid          bool      `json:"valid"`
	ManifestDigest string    `json:"manifest_digest"`
	AuditChainHead string    `json:"audit_chain_head"`
	EventCount     int64     `json:"event_count"`
	CheckedAt      time.Time `json:"checked_at"`
	Problems       []string  `json:"problems,omitempty"`
}
type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
