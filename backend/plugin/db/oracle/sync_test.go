package oracle

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// corruptedVietnamese simulates Vietnamese text encoded in Windows-1258
// that was inserted into an AL32UTF8 Oracle database via a misconfigured client.
// 0xe1 is 'á' in Win-1258 but an invalid UTF-8 lead byte without proper continuation.
var corruptedVietnamese = "Tr\xe1ng th\xe1i"

func TestOracleDefinitionSanitizedUTF8MarshalSuccess(t *testing.T) {
	var logBuf bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(originalLogger)
	corruptedDefinition := corruptedVietnamese + "\nSELECT 1 FROM DUAL"

	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "TESTDB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "TEST_SCHEMA",
				Views: []*storepb.ViewMetadata{
					{
						Name: "V_CUSTOMER",
						Definition: sanitizeOracleDefinition(
							"TEST_SCHEMA",
							"VIEW",
							"V_CUSTOMER",
							corruptedDefinition,
						),
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name: "MV_CUSTOMER",
						Definition: sanitizeOracleDefinition(
							"TEST_SCHEMA",
							"MATERIALIZED VIEW",
							"MV_CUSTOMER",
							corruptedDefinition,
						),
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name: "FN_VALIDATE",
						Definition: sanitizeOracleDefinition(
							"TEST_SCHEMA",
							"FUNCTION",
							"FN_VALIDATE",
							corruptedDefinition,
						),
					},
				},
				Procedures: []*storepb.ProcedureMetadata{
					{
						Name: "PR_VALIDATE",
						Definition: sanitizeOracleDefinition(
							"TEST_SCHEMA",
							"PROCEDURE",
							"PR_VALIDATE",
							corruptedDefinition,
						),
					},
				},
				Packages: []*storepb.PackageMetadata{
					{
						Name: "PKG_VALIDATE",
						Definition: sanitizeOracleDefinition(
							"TEST_SCHEMA",
							"PACKAGE",
							"PKG_VALIDATE",
							corruptedDefinition,
						),
					},
				},
			},
		},
	}

	_, err := protojson.Marshal(metadata)
	require.NoError(t, err, "marshal must succeed after definition sanitization")
	require.Contains(t, metadata.Schemas[0].Views[0].Definition, "\\xe1")
	require.Contains(t, metadata.Schemas[0].Views[0].Definition, "\nSELECT 1 FROM DUAL")

	logText := logBuf.String()
	require.Contains(t, logText, "sanitized invalid UTF-8 in Oracle metadata")
	require.Contains(t, logText, "schema=TEST_SCHEMA")
	require.Contains(t, logText, "object_type=VIEW")
	require.Contains(t, logText, "object_name=V_CUSTOMER")
	require.Contains(t, logText, "field=definition")
}

func TestOracleTriggerBodySanitizedUTF8MarshalSuccess(t *testing.T) {
	var logBuf bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(originalLogger)

	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "TESTDB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "TEST_SCHEMA",
				Tables: []*storepb.TableMetadata{
					{
						Name: "CUSTOMER",
						Triggers: []*storepb.TriggerMetadata{
							{
								Name: "TRG_CUSTOMER",
								Body: sanitizeOracleMetadataString(
									"TEST_SCHEMA",
									"TRIGGER",
									"TRG_CUSTOMER",
									"body",
									constructTriggerBody(
										"TRG_CUSTOMER BEFORE INSERT ON CUSTOMER\n",
										corruptedVietnamese+"\nBEGIN NULL; END;",
									),
								),
							},
						},
					},
				},
			},
		},
	}

	_, err := protojson.Marshal(metadata)
	require.NoError(t, err, "marshal must succeed after trigger body sanitization")
	require.Contains(t, metadata.Schemas[0].Tables[0].Triggers[0].Body, "\\xe1")
	require.Contains(t, metadata.Schemas[0].Tables[0].Triggers[0].Body, "\nBEGIN NULL; END;")

	logText := logBuf.String()
	require.Contains(t, logText, "sanitized invalid UTF-8 in Oracle metadata")
	require.Contains(t, logText, "schema=TEST_SCHEMA")
	require.Contains(t, logText, "object_type=TRIGGER")
	require.Contains(t, logText, "object_name=TRG_CUSTOMER")
	require.Contains(t, logText, "field=body")
}

// fillAllStringFields recursively sets every string field reachable from m
// (including nested messages, repeated fields, and map keys/values) to the
// given value, instantiating one element for each nested message, list, and
// map so that no string field in the schema shape stays unvisited.
