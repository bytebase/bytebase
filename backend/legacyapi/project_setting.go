package api

import (
	"encoding/json"
	"errors"
)

// LGTMCheckValue is the type of the LGTM check value.
type LGTMCheckValue string

const (
	// LGTMValueDisabled means no LGTM check.
	LGTMValueDisabled LGTMCheckValue = "DISABLED"
	// LGTMValueProjectOwner means check LGTM from project owners.
	LGTMValueProjectOwner LGTMCheckValue = "PROJECT_OWNER"
	// LGTMValueProjectMember means check LGTM from project members.
	LGTMValueProjectMember LGTMCheckValue = "PROJECT_MEMBER"
)

// LGTMCheckSetting is the setting of LGTM check.
type LGTMCheckSetting struct {
	Value LGTMCheckValue `json:"value" jsonapi:"attr,value"`
}

// GetDefaultLGTMCheckSetting returns the default LGTM check setting.
func GetDefaultLGTMCheckSetting() LGTMCheckSetting {
	return LGTMCheckSetting{
		Value: LGTMValueDisabled,
	}
}

// Scan implements database/sql Scanner interface, converts JSONB to LGTMCheckSetting struct.
func (s *LGTMCheckSetting) Scan(src interface{}) error {
	if bs, ok := src.([]byte); ok {
		if string(bs) == "{}" {
			// handle '{}', return default values
			*s = GetDefaultLGTMCheckSetting()
			return nil
		}
		return json.Unmarshal(bs, s)
	}
	return errors.New("failed to scan lgtm_check")
}
