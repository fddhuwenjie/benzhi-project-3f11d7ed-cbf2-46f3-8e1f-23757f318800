package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type checkClient struct {
	base     string
	client   *http.Client
	revision int64
	caseID   string
}

func runSelfCheck(c config, log *slog.Logger) error {
	temp, err := os.MkdirTemp("", "conservation-self-check-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	c.dataDir = temp
	rt, err := buildRuntime(c, log)
	if err != nil {
		return err
	}
	done := rt.serve()
	cc := &checkClient{base: "http://" + rt.listener.Addr().String(), client: &http.Client{Timeout: 5 * time.Second}}
	checkErr := cc.run()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	closeErr := rt.close(ctx)
	serveErr := <-done
	if checkErr != nil {
		return checkErr
	}
	if closeErr != nil {
		return closeErr
	}
	if serveErr != nil {
		return serveErr
	}
	fmt.Println("自检通过：覆盖预检、方案历史、试验门禁、执行进度、稳定性趋势、放行与证据封存均正常")
	return nil
}
func (c *checkClient) request(method, path string, body any, want int) (map[string]any, http.Header, error) {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return nil, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != want {
		return nil, resp.Header, fmt.Errorf("%s %s: 期望 %d，实际 %d: %s", method, path, want, resp.StatusCode, string(raw))
	}
	var out map[string]any
	if len(raw) > 0 {
		if err = json.Unmarshal(raw, &out); err != nil {
			return nil, resp.Header, err
		}
	}
	return out, resp.Header, nil
}
func wc(id string, revision int64, actor, role string) map[string]any {
	return map[string]any{"request_id": id, "expected_revision": revision, "actor_id": actor, "role": role}
}
func merge(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
func (c *checkClient) command(path string, body map[string]any, want int) (map[string]any, error) {
	out, _, err := c.request("POST", path, body, want)
	if err == nil && want < 300 {
		if rev, ok := out["revision"].(float64); ok {
			c.revision = int64(rev)
		}
	}
	return out, err
}
func (c *checkClient) run() error {
	create := merge(wc("sc-create", 0, "conservator-1", "conservator"), map[string]any{"manuscript_code": "MS-SC-001", "title": "自检手稿", "custodian_id": "custodian-1", "significance_note": "具有重要历史价值", "treatment_goal": "稳定脆弱纸张", "initial_risk": "颜料遇湿迁移", "required_regions": []map[string]string{{"leaf_ref": "1r", "region_ref": "center"}}})
	out, header, err := c.request("POST", "/api/v1/conservation-cases", create, 201)
	if err != nil {
		return err
	}
	c.caseID = out["id"].(string)
	c.revision = int64(out["revision"].(float64))
	replayed, rh, err := c.request("POST", "/api/v1/conservation-cases", create, 201)
	if err != nil {
		return err
	}
	if replayed["id"] != c.caseID || rh.Get("Idempotent-Replay") != "true" {
		return fmt.Errorf("创建幂等重放失败")
	}
	_ = header
	path := "/api/v1/conservation-cases/" + c.caseID
	if _, err = c.command(path+"/conditions", merge(wc("sc-condition", c.revision, "conservator-1", "conservator"), map[string]any{"leaf_ref": "1r", "region_ref": "center", "medium": "铁胆墨", "damage_type": "脆化", "severity": 3, "measurement": "裂口 2mm", "evidence_ref": "sha256:condition"}), 200); err != nil {
		return err
	}
	if coverage, _, getErr := c.request("GET", path+"/conditions/coverage", nil, 200); getErr != nil {
		return getErr
	} else if coverage["coverage_percentage"] != float64(100) {
		return fmt.Errorf("状况覆盖报告无效")
	}
	if _, err = c.command(path+"/baseline-lock", wc("sc-lock", c.revision, "conservator-1", "conservator"), 200); err != nil {
		return err
	}
	step := map[string]any{"index": 1, "purpose": "加固裂口", "material": "日本纸", "parameters": map[string]float64{"moisture": 5}, "tolerances": map[string]float64{"moisture": 0.5}, "reversibility": "可用受控湿度移除", "stop_condition": "墨迹色差大于 1", "risk_mitigation": "局部低湿操作"}
	plan := merge(wc("sc-plan", c.revision, "conservator-1", "conservator"), map[string]any{"steps": []any{step}, "reversibility_note": "所有粘接均可逆", "trace_preservation_note": "保留历史修补痕迹", "risk_controls": "隔离测试并设置停止点"})
	if _, err = c.command(path+"/plans", plan, 200); err != nil {
		return err
	}
	if _, err = c.command(path+"/plan-submit", wc("sc-submit", c.revision, "conservator-1", "conservator"), 200); err != nil {
		return err
	}
	if _, _, err = c.request("GET", path+"/plans", nil, 200); err != nil {
		return err
	}
	max := 1.0
	trial := merge(wc("sc-trial", c.revision, "conservator-1", "conservator"), map[string]any{"plan_version": 1, "material_code": "JP-KOZO", "protocol": "局部加速老化", "thresholds": []any{map[string]any{"name": "delta_e", "max": max}}, "measurements": []any{map[string]any{"name": "delta_e", "value": 0.4}}, "evidence_ref": "sha256:trial"})
	if _, err = c.command(path+"/trials", trial, 200); err != nil {
		return err
	}
	if trials, _, getErr := c.request("GET", path+"/trials", nil, 200); getErr != nil {
		return getErr
	} else if trials["current_gate_status"] != "passed" {
		return fmt.Errorf("材料试验门禁汇总无效")
	}
	if _, err = c.command(path+"/ethics-review", merge(wc("sc-ethics", c.revision, "custodian-1", "custodian"), map[string]any{"decision": "approve", "reason": "符合最小干预和授权范围"}), 200); err != nil {
		return err
	}
	stale := merge(wc("sc-stale", c.revision-1, "conservator-1", "conservator"), map[string]any{"step_index": 1, "actual_parameters": map[string]float64{"moisture": 5}, "evidence_ref": "sha256:stale"})
	if _, err = c.command(path+"/checkpoints", stale, 409); err != nil {
		return err
	}
	deviation := merge(wc("sc-checkpoint", c.revision, "conservator-1", "conservator"), map[string]any{"step_index": 1, "actual_parameters": map[string]float64{"moisture": 6}, "evidence_ref": "sha256:checkpoint"})
	paused, err := c.command(path+"/checkpoints", deviation, 200)
	if err != nil {
		return err
	}
	if paused["state"] != "paused" {
		return fmt.Errorf("参数偏离未自动暂停")
	}
	if status, _, getErr := c.request("GET", path+"/execution-status", nil, 200); getErr != nil {
		return getErr
	} else if status["state"] != "paused" {
		return fmt.Errorf("处理进度未反映暂停状态")
	}
	resolve := merge(wc("sc-resolve", c.revision, "custodian-1", "custodian"), map[string]any{"impact": "未发现墨迹迁移", "remediation": "降低湿度并延长干燥", "verified_by": "reviewer-remediation"})
	if _, err = c.command(path+"/deviation-resolution", resolve, 200); err != nil {
		return err
	}
	warp := 0.3
	color := 1.0
	stability := merge(wc("sc-stability", c.revision, "conservator-2", "conservator"), map[string]any{"duration_hours": 48, "thresholds": []any{map[string]any{"name": "warping_mm", "max": warp}, map[string]any{"name": "delta_e", "max": color}}, "measurements": []any{map[string]any{"name": "warping_mm", "value": 0.1}, map[string]any{"name": "delta_e", "value": 0.2}}, "evidence_ref": "sha256:stability"})
	if _, err = c.command(path+"/stability", stability, 200); err != nil {
		return err
	}
	if report, _, getErr := c.request("GET", path+"/stability/report", nil, 200); getErr != nil {
		return getErr
	} else if report["cumulative_hours"] != float64(48) {
		return fmt.Errorf("稳定性趋势报告无效")
	}
	if _, err = c.command(path+"/release", merge(wc("sc-release", c.revision, "independent-reviewer", "reviewer"), map[string]any{"statement": "独立核验处理记录与稳定性指标，准予放行"}), 200); err != nil {
		return err
	}
	if _, err = c.command(path+"/archive", wc("sc-archive", c.revision, "custodian-1", "custodian"), 200); err != nil {
		return err
	}
	if _, _, err = c.request("POST", path+"/archive-verification", nil, 200); err != nil {
		return err
	}
	if _, err = c.command(path+"/baseline-lock", wc("sc-after-archive", c.revision, "conservator-1", "conservator"), 409); err != nil {
		return err
	}
	if _, _, err = c.request("GET", path+"/archive", nil, 200); err != nil {
		return err
	}
	return nil
}
