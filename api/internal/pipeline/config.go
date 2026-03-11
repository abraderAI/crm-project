// Package pipeline provides CRM sales pipeline management with configurable
// stages, transition validation, and per-org customization.
package pipeline

import (
	"encoding/json"
	"fmt"
)

// Stage represents a pipeline stage name.
type Stage string

// Default pipeline stages.
const (
	StageNewLead     Stage = "new_lead"
	StageContacted   Stage = "contacted"
	StageQualified   Stage = "qualified"
	StageProposal    Stage = "proposal"
	StageNegotiation Stage = "negotiation"
	StageClosedWon   Stage = "closed_won"
	StageClosedLost  Stage = "closed_lost"
	StageNurturing   Stage = "nurturing"
)

// StageInfo describes a pipeline stage for API responses.
type StageInfo struct {
	Name        Stage   `json:"name"`
	Label       string  `json:"label"`
	Order       int     `json:"order"`
	IsFinal     bool    `json:"is_final"`
	Transitions []Stage `json:"transitions"`
}

// DefaultStages returns the ordered default pipeline stages.
func DefaultStages() []StageInfo {
	return []StageInfo{
		{Name: StageNewLead, Label: "New Lead", Order: 0, IsFinal: false, Transitions: []Stage{StageContacted, StageNurturing, StageClosedLost}},
		{Name: StageContacted, Label: "Contacted", Order: 1, IsFinal: false, Transitions: []Stage{StageQualified, StageNurturing, StageClosedLost}},
		{Name: StageQualified, Label: "Qualified", Order: 2, IsFinal: false, Transitions: []Stage{StageProposal, StageNurturing, StageClosedLost}},
		{Name: StageProposal, Label: "Proposal", Order: 3, IsFinal: false, Transitions: []Stage{StageNegotiation, StageNurturing, StageClosedLost}},
		{Name: StageNegotiation, Label: "Negotiation", Order: 4, IsFinal: false, Transitions: []Stage{StageClosedWon, StageClosedLost, StageNurturing, StageProposal}},
		{Name: StageClosedWon, Label: "Closed Won", Order: 5, IsFinal: true, Transitions: []Stage{}},
		{Name: StageClosedLost, Label: "Closed Lost", Order: 6, IsFinal: true, Transitions: []Stage{StageNurturing}},
		{Name: StageNurturing, Label: "Nurturing", Order: 7, IsFinal: false, Transitions: []Stage{StageContacted, StageQualified, StageClosedLost}},
	}
}

// Config holds pipeline configuration for an org.
type Config struct {
	Stages []StageInfo `json:"stages"`
}

// DefaultConfig returns the default pipeline configuration.
func DefaultConfig() *Config {
	return &Config{Stages: DefaultStages()}
}

// TransitionMap builds a map of allowed transitions from a stage list.
func TransitionMap(stages []StageInfo) map[Stage][]Stage {
	m := make(map[Stage][]Stage, len(stages))
	for _, s := range stages {
		m[s.Name] = s.Transitions
	}
	return m
}

// ValidateTransition checks if transitioning from current to next is allowed.
func ValidateTransition(stages []StageInfo, current, next Stage) error {
	if current == "" {
		// No current stage — can only move to the first stage or nurturing.
		if next == StageNewLead || next == StageNurturing {
			return nil
		}
		return fmt.Errorf("initial stage must be %s or %s, got %s", StageNewLead, StageNurturing, next)
	}

	tm := TransitionMap(stages)
	allowed, ok := tm[current]
	if !ok {
		return fmt.Errorf("unknown current stage: %s", current)
	}

	for _, a := range allowed {
		if a == next {
			return nil
		}
	}

	return fmt.Errorf("transition from %s to %s is not allowed", current, next)
}

// IsValidStage checks if a stage name is valid in the given config.
func IsValidStage(stages []StageInfo, stage Stage) bool {
	for _, s := range stages {
		if s.Name == stage {
			return true
		}
	}
	return false
}

// IsFinalStage checks if a stage is a terminal stage.
func IsFinalStage(stages []StageInfo, stage Stage) bool {
	for _, s := range stages {
		if s.Name == stage {
			return s.IsFinal
		}
	}
	return false
}

// ParseConfigFromMetadata extracts pipeline config from org metadata JSON.
// Returns nil if no custom config is found (caller should use defaults).
func ParseConfigFromMetadata(metadataJSON string) *Config {
	if metadataJSON == "" || metadataJSON == "{}" {
		return nil
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil
	}

	raw, ok := meta["pipeline_config"]
	if !ok {
		return nil
	}

	configBytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var cfg Config
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return nil
	}
	if len(cfg.Stages) == 0 {
		return nil
	}

	return &cfg
}
