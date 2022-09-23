package collector

import "github.com/bytebase/bytebase/plugin/advisor"

type CollectionFlag int

const (
	CollectionFlagCreateTable  CollectionFlag = 0b1
	CollectionFlagAddColumn    CollectionFlag = 0b10
	CollectionFlagChangeColumn CollectionFlag = 0b100
	CollectionFlagModifyColumn CollectionFlag = 0b1000
	CollectionFlagAlterColumn  CollectionFlag = 0b10000
	CollectionFlagRenameColumn CollectionFlag = 0b100000
)

type CollectionContext struct {
	Flag  CollectionFlag
	Level advisor.Status

	// catalogCollector specific fields.
	Replace bool
}

func (flag CollectionFlag) hasFlag(target CollectionFlag) bool {
	return flag&target == target
}
