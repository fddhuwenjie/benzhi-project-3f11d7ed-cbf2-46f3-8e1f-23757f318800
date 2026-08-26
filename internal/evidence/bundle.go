package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"manuscript-conservation-gate/internal/domain"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Manager struct {
	dir string
	now func() time.Time
}
type document struct {
	Name    string          `json:"name"`
	SHA256  string          `json:"sha256"`
	Content json.RawMessage `json:"content"`
}
type bundle struct {
	Schema         string              `json:"schema"`
	CaseID         string              `json:"case_id"`
	CaseRevision   int64               `json:"case_revision"`
	GeneratedAt    time.Time           `json:"generated_at"`
	Documents      []document          `json:"documents"`
	Events         []domain.AuditEvent `json:"events"`
	EventCount     int64               `json:"event_count"`
	AuditChainHead string              `json:"audit_chain_head"`
	ManifestDigest string              `json:"manifest_digest"`
}

func New(dir string) *Manager {
	return &Manager{dir: dir, now: func() time.Time { return time.Now().UTC() }}
}
func digest(b []byte) string             { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func canonical(v any) ([]byte, error)    { return json.Marshal(v) }
func (m *Manager) path(id string) string { return filepath.Join(m.dir, id+".evidence.json") }
func (m *Manager) Generate(ctx context.Context, item *domain.ConservationCase, events []domain.AuditEvent, head string) (domain.ArchiveManifest, error) {
	select {
	case <-ctx.Done():
		return domain.ArchiveManifest{}, ctx.Err()
	default:
	}
	snapshot, err := canonical(item)
	if err != nil {
		return domain.ArchiveManifest{}, err
	}
	plans, _ := canonical(item.Plans)
	trials, _ := canonical(item.Trials)
	checkpoints, _ := canonical(item.Checkpoints)
	release, _ := canonical(item.Release)
	docs := []document{{"case_snapshot.json", digest(snapshot), snapshot}, {"checkpoints.json", digest(checkpoints), checkpoints}, {"plans.json", digest(plans), plans}, {"release.json", digest(release), release}, {"trials.json", digest(trials), trials}}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	b := bundle{Schema: "manuscript-conservation-evidence/v1", CaseID: item.ID, CaseRevision: item.Revision, GeneratedAt: m.now(), Documents: docs, Events: events, EventCount: int64(len(events)), AuditChainHead: head}
	unsigned, err := canonical(b)
	if err != nil {
		return domain.ArchiveManifest{}, err
	}
	b.ManifestDigest = digest(unsigned)
	final, err := canonical(b)
	if err != nil {
		return domain.ArchiveManifest{}, err
	}
	if err = m.atomicWrite(m.path(item.ID), final); err != nil {
		return domain.ArchiveManifest{}, err
	}
	digests := map[string]string{}
	for _, d := range docs {
		digests[d.Name] = d.SHA256
	}
	return domain.ArchiveManifest{CaseID: item.ID, CaseRevision: item.Revision, GeneratedAt: b.GeneratedAt, DocumentDigests: digests, EventCount: b.EventCount, AuditChainHead: head, ManifestDigest: b.ManifestDigest}, nil
}
func (m *Manager) atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(m.dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(m.dir, ".evidence-*")
	if err != nil {
		return err
	}
	name := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	dir, err := os.Open(m.dir)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err = dir.Sync(); err != nil {
		return fmt.Errorf("同步证据目录: %w", err)
	}
	ok = true
	return nil
}
func (m *Manager) Read(ctx context.Context, id string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return os.ReadFile(m.path(id))
}
