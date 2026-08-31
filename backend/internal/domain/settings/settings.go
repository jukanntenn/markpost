// Package settings defines the runtime settings domain: operational strategy
// switches that admins flip without a deploy (MRFC 2026-08-23-github-login-vip-grant-strategy).
package settings

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// KeyVIP names the GitHub-login VIP grant strategy switch. The seed row ships
// enabled so the strategy launches on.
const KeyVIP = "vip"

// KeyVIPRetention names the VIP-class retention default materialized onto a
// user at grant time (MRFC 2026-08-31-per-user-history-retention-policy).
const KeyVIPRetention = "vip_retention_days"

// SettingValue is the JSONB payload of a settings row. v1 carries a single
// enabled switch; the vip_retention_days strategy extends the struct with a
// Days pointer — the schema stays untouched, each key reads the shape it owns.
type SettingValue struct {
	Enabled bool `json:"enabled"`
	// Days carries the vip_retention_days value: nil follows the global
	// default, 0 keeps forever, 1–3650 keeps N days.
	Days *int `json:"days,omitempty"`
}

// Value serializes for the jsonb column. A string (not []byte) is returned:
// the postgres driver maps []byte to bytea, which a jsonb column rejects.
func (v SettingValue) Value() (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("settings value marshal: %w", err)
	}
	return string(b), nil
}

// Scan deserializes the jsonb column (postgres delivers it as []byte or string).
func (v *SettingValue) Scan(src any) error {
	switch s := src.(type) {
	case nil:
		*v = SettingValue{}
	case []byte:
		if err := json.Unmarshal(s, v); err != nil {
			return fmt.Errorf("settings value scan: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(s), v); err != nil {
			return fmt.Errorf("settings value scan: %w", err)
		}
	default:
		return fmt.Errorf("settings value scan: unsupported type %T", src)
	}
	return nil
}

// Setting is one keyed runtime setting row.
type Setting struct {
	Key       string       `json:"key" gorm:"primaryKey;column:key"`
	Value     SettingValue `json:"value" gorm:"not null;type:jsonb;column:value"`
	UpdatedBy *int64       `json:"updated_by" gorm:"column:updated_by"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}
