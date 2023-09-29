package base

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	DefaultMaskingLevel storepb.MaskingLevel = storepb.MaskingLevel_NONE
	MaxMaskingLevel     storepb.MaskingLevel = storepb.MaskingLevel_FULL
)

// FieldInfo is the masking field info.
type FieldInfo struct {
	Name         string
	Table        string
	Schema       string
	Database     string
	MaskingLevel storepb.MaskingLevel
}
