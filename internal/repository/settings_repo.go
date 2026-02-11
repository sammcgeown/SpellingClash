package repository

import (
	"spellingclash/internal/database"
)

type SettingsRepository struct {
	db *database.DB
}

func NewSettingsRepository(db *database.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetSetting retrieves a setting value by key
func (r *SettingsRepository) GetSetting(key string) (string, error) {
	var value string
	query := `SELECT value FROM settings WHERE key = ?`
	err := r.db.QueryRow(query, key).Scan(&value)
	return value, err
}

// SetSetting updates or inserts a setting
func (r *SettingsRepository) SetSetting(key, value string) error {
	query := r.db.Dialect.UpsertSettings()
	_, err := r.db.Exec(query, key, value)
	return err
}

// IsInviteOnlyMode checks if invite-only mode is enabled
func (r *SettingsRepository) IsInviteOnlyMode() bool {
	value, err := r.GetSetting("invite_only_mode")
	if err != nil {
		return false // Default to open registration
	}
	return value == "true"
}

// SetInviteOnlyMode enables or disables invite-only mode
func (r *SettingsRepository) SetInviteOnlyMode(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return r.SetSetting("invite_only_mode", value)
}
