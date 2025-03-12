package redshift

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_REDSHIFT, GetDesignSchema)
}

func GetDesignSchema(to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)

	var sb strings.Builder

	if err := writeTables(&sb, to, toState); err != nil {
		return "", err
	}
	if err := writeViews(&sb, to, toState); err != nil {
		return "", err
	}

	s := sb.String()
	// Make goyamlv3 happy.
	s = strings.TrimLeft(s, "\n")
	return s, nil
}
