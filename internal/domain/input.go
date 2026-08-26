package domain

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

func ValidateIdentifier(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Invalid(field, "不能为空")
	}
	if len(value) > 128 {
		return Invalid(field, "长度不能超过 128 字节")
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return Invalid(field, "不得包含空白或控制字符")
		}
	}
	return nil
}
func ValidateNarrative(field, value string, max int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Invalid(field, "不能为空")
	}
	if len(value) > max {
		return Invalid(field, fmt.Sprintf("长度不能超过 %d 字节", max))
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return Invalid(field, "包含不允许的控制字符")
		}
	}
	return nil
}
func ValidateEvidenceRef(value string) error {
	if err := ValidateNarrative("evidence_ref", value, 512); err != nil {
		return err
	}
	if strings.ContainsAny(value, "\r\n") {
		return Invalid("evidence_ref", "必须为单行引用")
	}
	return nil
}
func ValidateParameters(field string, values map[string]float64) error {
	if len(values) == 0 {
		return Invalid(field, "至少包含一个参数")
	}
	for name, value := range values {
		if strings.TrimSpace(name) == "" {
			return Invalid(field, "参数名称不能为空")
		}
		if len(name) > 128 {
			return Invalid(field, "参数名称过长")
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return Invalid(field, "参数必须为有限数值")
		}
	}
	return nil
}
func ValidateMetrics(rules []MetricRule, values []MetricValue) error {
	ruleNames := map[string]bool{}
	for i, r := range rules {
		if strings.TrimSpace(r.Name) == "" {
			return Invalid(fmt.Sprintf("thresholds[%d].name", i), "不能为空")
		}
		if ruleNames[r.Name] {
			return Invalid("thresholds", "指标重复")
		}
		ruleNames[r.Name] = true
		if r.Min == nil && r.Max == nil {
			return Invalid(fmt.Sprintf("thresholds[%d]", i), "至少声明一个边界")
		}
		if r.Min != nil && (math.IsNaN(*r.Min) || math.IsInf(*r.Min, 0)) {
			return Invalid(fmt.Sprintf("thresholds[%d].min", i), "必须为有限数值")
		}
		if r.Max != nil && (math.IsNaN(*r.Max) || math.IsInf(*r.Max, 0)) {
			return Invalid(fmt.Sprintf("thresholds[%d].max", i), "必须为有限数值")
		}
		if r.Min != nil && r.Max != nil && *r.Min > *r.Max {
			return Invalid(fmt.Sprintf("thresholds[%d]", i), "下限不得高于上限")
		}
	}
	valueNames := map[string]bool{}
	for i, v := range values {
		if strings.TrimSpace(v.Name) == "" {
			return Invalid(fmt.Sprintf("measurements[%d].name", i), "不能为空")
		}
		if valueNames[v.Name] {
			return Invalid("measurements", "指标重复")
		}
		valueNames[v.Name] = true
		if math.IsNaN(v.Value) || math.IsInf(v.Value, 0) {
			return Invalid(fmt.Sprintf("measurements[%d].value", i), "必须为有限数值")
		}
		if !ruleNames[v.Name] {
			return Invalid(fmt.Sprintf("measurements[%d].name", i), "没有对应阈值")
		}
	}
	return nil
}
