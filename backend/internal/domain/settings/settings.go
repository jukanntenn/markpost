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

// SettingValue is the JSONB payload of a settings row. v1 carries a single
// enabled switch; future strategies extend the struct rather than the schema.
type SettingValue struct {
	Enabled bool `json:"enabled"`
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
