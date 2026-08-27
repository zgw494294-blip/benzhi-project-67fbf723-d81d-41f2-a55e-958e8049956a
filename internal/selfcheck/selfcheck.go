package selfcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Runner struct {
	Base   string
	Client *http.Client
}

func (r Runner) Run(ctx context.Context) error {
	c := r.Client
	if c == nil {
		c = &http.Client{Timeout: 2 * time.Second}
	}
	post := func(path string, v any, key ...string) (map[string]any, error) {
		b, _ := json.Marshal(v)
		req, _ := http.NewRequestWithContext(ctx, "POST", r.Base+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if len(key) > 0 {
			req.Header.Set("Idempotency-Key", key[0])
		}
		resp, e := c.Do(req)
		if e != nil {
			return nil, e
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%v", out["error"])
		}
		return out, nil
	}
	id := fmt.Sprintf("selfcheck-%d", time.Now().UnixNano())
	created, e := post("/api/trials", map[string]any{"trialId": id, "speciesName": "自检植物", "accessionCode": "SC01", "collectionBatch": id, "replicateCount": 2, "seedsPerReplicate": 10, "actor": "selfcheck"})
	if e != nil {
		return e
	}
	revision := int64(created["revision"].(float64))
	protocol := map[string]any{"stratificationDays": 0, "observationDays": 2, "temperatureCelsius": 25, "lightRegime": "16h", "substrate": "培养基", "germinationThresholdPercent": 50}
	preview, e := post("/api/trials/"+id+"/protocol-preview", protocol)
	if e != nil {
		return e
	}
	protocol["contentDigest"] = preview["contentDigest"]
	protocol["expectedRevision"] = revision
	protocol["actor"] = "selfcheck"
	locked, e := post("/api/trials/"+id+"/protocol", protocol, "selfcheck-protocol")
	if e != nil {
		return e
	}
	revision = int64(locked["revision"].(float64))
	started, e := post("/api/trials/"+id+"/start", map[string]any{"expectedRevision": revision, "actor": "selfcheck"}, "selfcheck-start")
	if e != nil {
		return e
	}
	revision = int64(started["revision"].(float64))
	for d := 1; d <= 2; d++ {
		observations := []map[string]any{}
		for rep := 1; rep <= 2; rep++ {
			observations = append(observations, map[string]any{"replicateIndex": rep, "newlyGerminated": 5, "newlyNonviable": 0, "temperatureCelsius": 25, "recordedBy": "selfcheck"})
		}
		observed, e := post(fmt.Sprintf("/api/trials/%s/observe", id), map[string]any{"expectedRevision": revision, "dayIndex": d, "observations": observations, "actor": "selfcheck"}, fmt.Sprintf("selfcheck-observe-%d", d))
		if e != nil {
			return e
		}
		revision = int64(observed["revision"].(float64))
	}
	submitted, e := post("/api/trials/"+id+"/submit", map[string]any{"expectedRevision": revision, "actor": "selfcheck"}, "selfcheck-submit")
	if e != nil {
		return e
	}
	revision = int64(submitted["revision"].(float64))
	packages := submitted["reviewPackages"].([]any)
	current := packages[len(packages)-1].(map[string]any)
	out, e := post("/api/trials/"+id+"/review", map[string]any{"expectedRevision": revision, "decision": "APPROVE", "reviewer": "selfcheck", "conclusion": "自检数据符合方案要求", "snapshotDigest": current["snapshotDigest"]}, "selfcheck-review")
	if e != nil {
		return e
	}
	if out["status"] != "CLOSED" {
		return fmt.Errorf("状态未关闭")
	}
	return nil
}
