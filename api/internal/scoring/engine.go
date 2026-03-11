// Package scoring provides a rule-based lead scoring engine for CRM threads.
// Rules are config-driven with per-org customization via org metadata.
package scoring

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ScoringRule defines a condition-based scoring rule.
type ScoringRule struct {
	Name     string `json:"name"`
	Path     string `json:"path"`     // JSON path in metadata, e.g., "stage", "priority".
	Operator string `json:"operator"` // "eq", "gt", "gte", "lt", "lte", "contains", "exists".
	Value    string `json:"value"`    // Expected value for comparison.
	Points   int    `json:"points"`   // Points to add if rule matches.
}

// ScoreBreakdown shows per-rule scores.
type ScoreBreakdown struct {
	TotalScore int            `json:"total_score"`
	Rules      []RuleResult   `json:"rules"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// RuleResult is the result of evaluating a single rule.
type RuleResult struct {
	RuleName string `json:"rule_name"`
	Matched  bool   `json:"matched"`
	Points   int    `json:"points"`
}

// DefaultRules returns the default scoring rules.
func DefaultRules() []ScoringRule {
	return []ScoringRule{
		{Name: "stage_new_lead", Path: "stage", Operator: "eq", Value: "new_lead", Points: 5},
		{Name: "stage_contacted", Path: "stage", Operator: "eq", Value: "contacted", Points: 15},
		{Name: "stage_qualified", Path: "stage", Operator: "eq", Value: "qualified", Points: 30},
		{Name: "stage_proposal", Path: "stage", Operator: "eq", Value: "proposal", Points: 50},
		{Name: "stage_negotiation", Path: "stage", Operator: "eq", Value: "negotiation", Points: 75},
		{Name: "stage_closed_won", Path: "stage", Operator: "eq", Value: "closed_won", Points: 100},
		{Name: "high_priority", Path: "priority", Operator: "eq", Value: "high", Points: 20},
		{Name: "medium_priority", Path: "priority", Operator: "eq", Value: "medium", Points: 10},
		{Name: "has_company", Path: "company", Operator: "exists", Value: "", Points: 10},
		{Name: "has_email", Path: "contact_email", Operator: "exists", Value: "", Points: 5},
		{Name: "high_value", Path: "deal_value", Operator: "gt", Value: "10000", Points: 25},
		{Name: "medium_value", Path: "deal_value", Operator: "gt", Value: "1000", Points: 10},
	}
}

// Evaluate runs all rules against the given metadata and returns the breakdown.
func Evaluate(rules []ScoringRule, metadataJSON string) *ScoreBreakdown {
	var meta map[string]any
	if metadataJSON != "" && metadataJSON != "{}" {
		if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
			meta = make(map[string]any)
		}
	} else {
		meta = make(map[string]any)
	}

	breakdown := &ScoreBreakdown{
		Metadata: meta,
	}

	for _, rule := range rules {
		result := evaluateRule(rule, meta)
		breakdown.Rules = append(breakdown.Rules, result)
		if result.Matched {
			breakdown.TotalScore += result.Points
		}
	}

	return breakdown
}

// evaluateRule tests a single rule against metadata.
func evaluateRule(rule ScoringRule, meta map[string]any) RuleResult {
	result := RuleResult{
		RuleName: rule.Name,
		Points:   rule.Points,
	}

	val, exists := resolveMetaPath(meta, rule.Path)

	switch strings.ToLower(rule.Operator) {
	case "eq":
		result.Matched = exists && fmt.Sprintf("%v", val) == rule.Value
	case "exists":
		result.Matched = exists && val != nil && fmt.Sprintf("%v", val) != ""
	case "contains":
		if exists {
			result.Matched = strings.Contains(fmt.Sprintf("%v", val), rule.Value)
		}
	case "gt", "gte", "lt", "lte":
		if exists {
			result.Matched = compareNumeric(val, rule.Operator, rule.Value)
		}
	default:
		result.Matched = false
	}

	return result
}

// resolveMetaPath extracts a value from metadata by dotted path (e.g., "deal_value").
func resolveMetaPath(meta map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = meta
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// compareNumeric compares a metadata value against a threshold using the given operator.
func compareNumeric(val any, op, threshold string) bool {
	thresholdF, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return false
	}

	var valF float64
	switch v := val.(type) {
	case float64:
		valF = v
	case int:
		valF = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false
		}
		valF = parsed
	case json.Number:
		parsed, err := v.Float64()
		if err != nil {
			return false
		}
		valF = parsed
	default:
		return false
	}

	switch op {
	case "gt":
		return valF > thresholdF
	case "gte":
		return valF >= thresholdF
	case "lt":
		return valF < thresholdF
	case "lte":
		return valF <= thresholdF
	default:
		return false
	}
}

// ParseRulesFromMetadata extracts scoring rules from org metadata.
// Returns nil if no custom rules are found.
func ParseRulesFromMetadata(metadataJSON string) []ScoringRule {
	if metadataJSON == "" || metadataJSON == "{}" {
		return nil
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil
	}

	raw, ok := meta["scoring_rules"]
	if !ok {
		return nil
	}

	rulesBytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var rules []ScoringRule
	if err := json.Unmarshal(rulesBytes, &rules); err != nil {
		return nil
	}
	if len(rules) == 0 {
		return nil
	}

	return rules
}
