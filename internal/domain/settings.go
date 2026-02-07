package domain

import "time"

// Settings represents the application settings entity.
type Settings struct {
	ID                  string    `json:"id" db:"id"`
	RetentionPeriodDays int       `json:"retentionPeriodDays" db:"retention_period_days"`
	MaxExportSize       int       `json:"maxExportSize" db:"max_export_size"`
	UpdatedAt           time.Time `json:"updatedAt" db:"updated_at"`
	UpdatedBy           string    `json:"updatedBy" db:"updated_by"`
}

const (
	// GlobalSettingsID is the ID for the global settings record.
	GlobalSettingsID = "global"
)
