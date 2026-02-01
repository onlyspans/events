package dto

// SettingsDTO represents the settings data transfer object.
type SettingsDTO struct {
	RetentionPeriodDays int `json:"retentionPeriodDays"`
	MaxExportSize       int `json:"maxExportSize"`
}
