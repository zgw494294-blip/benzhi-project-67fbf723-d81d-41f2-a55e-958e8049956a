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

// missingFieldError wraps a response that was a valid JSON object on HTTP 2xx
// but lacked a required field (or had it in the wrong type). It lets the
// self-check fail through a controlled error path instead of panicking on a
// type assertion against a missing value.
type missingFieldError struct {
	field string
	path  string
}

func (e *missingFieldError) Error() string {
	if e.path != "" {
		return fmt.Sprintf("自检响应缺少必需字段 %q（%s）", e.field, e.path)
	}
	return fmt.Sprintf("自检响应缺少必需字段 %q", e.field)
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
		if e = json.NewDecoder(resp.Body).Decode(&out); e != nil {
			return nil, fmt.Errorf("自检响应解析失败（%s）: %w", path, e)
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%v", out["error"])
		}
		if out == nil {
			return nil, &missingFieldError{field: "响应体", path: path}
		}
		return out, nil
	}
	// requireInt extracts a required numeric "revision" field as int64. A
	// missing field or wrong type is reported as a diagnostic error rather
	// than allowing a type assertion to panic.
	requireInt := func(out map[string]any, field, path string) (int64, error) {
		v, ok := out[field]
		if !ok || v == nil {
			return 0, &missingFieldError{field: field, path: path}
		}
		f, ok := v.(float64)
		if !ok {
			return 0, &missingFieldError{field: field, path: path}
		}
		return int64(f), nil
	}
	// requireStr extracts a required string field, diagnosing its absence.
	requireStr := func(out map[string]any, field, path string) (string, error) {
		v, ok := out[field]
		if !ok || v == nil {
			return "", &missingFieldError{field: field, path: path}
		}
		s, ok := v.(string)
		if !ok {
			return "", &missingFieldError{field: field, path: path}
		}
		return s, nil
	}
	id := fmt.Sprintf("selfcheck-%d", time.Now().UnixNano())
	created, e := post("/api/trials", map[string]any{"trialId": id, "speciesName": "自检植物", "accessionCode": "SC01", "collectionBatch": id, "replicateCount": 2, "seedsPerReplicate": 10, "actor": "selfcheck"})
	if e != nil {
		return e
	}
	revision, e := requireInt(created, "revision", "/api/trials")
	if e != nil {
		return e
	}
	protocol := map[string]any{"stratificationDays": 0, "observationDays": 2, "temperatureCelsius": 25, "lightRegime": "16h", "substrate": "培养基", "germinationThresholdPercent": 50}
	preview, e := post("/api/trials/"+id+"/protocol-preview", protocol)
	if e != nil {
		return e
	}
	contentDigest, e := requireStr(preview, "contentDigest", "/api/trials/"+id+"/protocol-preview")
	if e != nil {
		return e
	}
	protocol["contentDigest"] = contentDigest
	protocol["expectedRevision"] = revision
	protocol["actor"] = "selfcheck"
	locked, e := post("/api/trials/"+id+"/protocol", protocol, "selfcheck-protocol")
	if e != nil {
		return e
	}
	revision, e = requireInt(locked, "revision", "/api/trials/"+id+"/protocol")
	if e != nil {
		return e
	}
	started, e := post("/api/trials/"+id+"/start", map[string]any{"expectedRevision": revision, "actor": "selfcheck"}, "selfcheck-start")
	if e != nil {
		return e
	}
	revision, e = requireInt(started, "revision", "/api/trials/"+id+"/start")
	if e != nil {
		return e
	}
	for d := 1; d <= 2; d++ {
		observations := []map[string]any{}
		for rep := 1; rep <= 2; rep++ {
			observations = append(observations, map[string]any{"replicateIndex": rep, "newlyGerminated": 5, "newlyNonviable": 0, "temperatureCelsius": 25, "recordedBy": "selfcheck"})
		}
		observePath := fmt.Sprintf("/api/trials/%s/observe", id)
		observed, e := post(observePath, map[string]any{"expectedRevision": revision, "dayIndex": d, "observations": observations, "actor": "selfcheck"}, fmt.Sprintf("selfcheck-observe-%d", d))
		if e != nil {
			return e
		}
		revision, e = requireInt(observed, "revision", observePath)
		if e != nil {
			return e
		}
	}
	submitPath := "/api/trials/" + id + "/submit"
	submitted, e := post(submitPath, map[string]any{"expectedRevision": revision, "actor": "selfcheck"}, "selfcheck-submit")
	if e != nil {
		return e
	}
	revision, e = requireInt(submitted, "revision", submitPath)
	if e != nil {
		return e
	}
	packages, e := requireSlice(submitted, "reviewPackages", submitPath)
	if e != nil {
		return e
	}
	if len(packages) == 0 {
		return &missingFieldError{field: "reviewPackages", path: submitPath}
	}
	last := packages[len(packages)-1]
	current, ok := last.(map[string]any)
	if !ok {
		return &missingFieldError{field: "reviewPackages[*]", path: submitPath}
	}
	snapshotDigest, e := requireStr(current, "snapshotDigest", submitPath)
	if e != nil {
		return e
	}
	reviewPath := "/api/trials/" + id + "/review"
	out, e := post(reviewPath, map[string]any{"expectedRevision": revision, "decision": "APPROVE", "reviewer": "selfcheck", "conclusion": "自检数据符合方案要求", "snapshotDigest": snapshotDigest}, "selfcheck-review")
	if e != nil {
		return e
	}
	status, _ := out["status"].(string)
	if status != "CLOSED" {
		return fmt.Errorf("状态未关闭")
	}
	return nil
}

// requireSlice extracts a required []any field, diagnosing its absence.
func requireSlice(out map[string]any, field, path string) ([]any, error) {
	v, ok := out[field]
	if !ok || v == nil {
		return nil, &missingFieldError{field: field, path: path}
	}
	s, ok := v.([]any)
	if !ok {
		return nil, &missingFieldError{field: field, path: path}
	}
	return s, nil
}
