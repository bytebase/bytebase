package oracle

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// corruptedVietnamese simulates Vietnamese text encoded in Windows-1258
// that was inserted into an AL32UTF8 Oracle database via a misconfigured client.
// 0xe1 is 'á' in Win-1258 but an invalid UTF-8 lead byte without proper continuation.
var corruptedVietnamese = "Tr\xe1ng th\xe1i"

func TestOracleCommentInvalidUTF8MarshalFailure(t *testing.T) {
	require.False(t, utf8.ValidString(corruptedVietnamese),
		"test precondition: input must be invalid UTF-8")

	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "TESTDB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "TEST_SCHEMA",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "CUSTOMER",
						Comment: corruptedVietnamese,
						Columns: []*storepb.ColumnMetadata{
							{
								Name:    "STATUS",
								Comment: corruptedVietnamese,
							},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:    "TRG_AUDIT",
								Comment: corruptedVietnamese,
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:    "V_CUSTOMER",
						Comment: corruptedVietnamese,
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:    "MV_CUSTOMER",
						Comment: corruptedVietnamese,
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name:    "SEQ_CUSTOMER",
						Comment: corruptedVietnamese,
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:    "FN_VALIDATE",
						Comment: corruptedVietnamese,
					},
				},
			},
		},
	}

	// Prove this is the exact failure the customer sees.
	_, err := protojson.Marshal(metadata)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid UTF-8")
}

func TestOracleCommentSanitizedUTF8MarshalSuccess(t *testing.T) {
	sanitized := common.SanitizeUTF8String(corruptedVietnamese)

	require.True(t, utf8.ValidString(sanitized),
		"sanitized string must be valid UTF-8")
	require.Contains(t, sanitized, "\\xe1",
		"sanitized string should preserve original bytes as hex")

	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "TESTDB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "TEST_SCHEMA",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "CUSTOMER",
						Comment: sanitized,
						Columns: []*storepb.ColumnMetadata{
							{
								Name:    "STATUS",
								Comment: sanitized,
							},
						},
						Triggers: []*storepb.TriggerMetadata{
							{
								Name:    "TRG_AUDIT",
								Comment: sanitized,
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:    "V_CUSTOMER",
						Comment: sanitized,
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:    "MV_CUSTOMER",
						Comment: sanitized,
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name:    "SEQ_CUSTOMER",
						Comment: sanitized,
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:    "FN_VALIDATE",
						Comment: sanitized,
					},
				},
			},
		},
	}

	_, err := protojson.Marshal(metadata)
	require.NoError(t, err, "marshal must succeed after sanitization")
}
