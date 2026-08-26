package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func HashEvent(previous string, sequence, revision int64, caseID, eventType, actor string, payload []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%d\n%s\n%d\n%s\n%s\n", previous, sequence, caseID, revision, eventType, actor)
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
func EventPayload(v any) []byte { b, _ := json.Marshal(v); return b }
