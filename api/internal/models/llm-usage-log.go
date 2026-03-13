package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LLMUsageLog tracks LLM enrichment API calls with token counts.
type LLMUsageLog struct {
	ID           string    `gorm:"type:text;primaryKey" json:"id"`
	Endpoint     string    `gorm:"type:text" json:"endpoint"`
	Model        string    `gorm:"type:text" json:"model"`
	InputTokens  int64     `gorm:"default:0" json:"input_tokens"`
	OutputTokens int64     `gorm:"default:0" json:"output_tokens"`
	DurationMs   int64     `gorm:"default:0" json:"duration_ms"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// BeforeCreate generates a UUIDv7 for the ID field if not already set.
func (l *LLMUsageLog) BeforeCreate(_ *gorm.DB) error {
	if l.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		l.ID = id.String()
	}
	return nil
}
