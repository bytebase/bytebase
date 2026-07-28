package oracle

import (
	"bytes"
	"log/slog"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

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

func TestOracleDefinitionInvalidUTF8MarshalFailure(t *testing.T) {
	require.False(t, utf8.ValidString(corruptedVietnamese),
		"test precondition: input must be invalid UTF-8")

	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "TESTDB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "TEST_SCHEMA",
				Views: []*storepb.ViewMetadata{
					{
						Name:       "V_CUSTOMER",
						Definition: corruptedVietnamese,
					},
				},
			},
		},
	}

	_, err := protojson.Marshal(metadata)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ViewMetadata.definition")
	require.Contains(t, err.Error(), "invalid UTF-8")
}

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
func fillAllStringFields(m protoreflect.Message, value string, depth int) {
	if depth <= 0 {
		return
	}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		switch {
		case fd.IsMap():
			mp := m.Mutable(fd).Map()
			var key protoreflect.MapKey
			if fd.MapKey().Kind() == protoreflect.StringKind {
				key = protoreflect.ValueOfString(value).MapKey()
			} else {
				key = protoreflect.ValueOfInt64(1).MapKey()
			}
			switch fd.MapValue().Kind() {
			case protoreflect.StringKind:
				mp.Set(key, protoreflect.ValueOfString(value))
			case protoreflect.MessageKind:
				nv := mp.NewValue()
				fillAllStringFields(nv.Message(), value, depth-1)
				mp.Set(key, nv)
			default:
				mp.Set(key, mp.NewValue())
			}
		case fd.IsList():
			list := m.Mutable(fd).List()
			switch fd.Kind() {
			case protoreflect.StringKind:
				list.Append(protoreflect.ValueOfString(value))
			case protoreflect.MessageKind:
				nv := list.NewElement()
				fillAllStringFields(nv.Message(), value, depth-1)
				list.Append(nv)
			default:
			}
		case fd.Kind() == protoreflect.StringKind:
			m.Set(fd, protoreflect.ValueOfString(value))
		case fd.Kind() == protoreflect.MessageKind:
			fillAllStringFields(m.Mutable(fd).Message(), value, depth-1)
		default:
		}
	}
}

// countInvalidStringFields walks m and returns how many string values
// (fields, list elements, map keys and values) hold invalid UTF-8.
func countInvalidStringFields(m protoreflect.Message) int {
	count := 0
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
				if fd.MapKey().Kind() == protoreflect.StringKind && !utf8.ValidString(k.String()) {
					count++
				}
				switch fd.MapValue().Kind() {
				case protoreflect.StringKind:
					if !utf8.ValidString(mv.String()) {
						count++
					}
				case protoreflect.MessageKind:
					count += countInvalidStringFields(mv.Message())
				default:
				}
				return true
			})
		case fd.IsList():
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				switch fd.Kind() {
				case protoreflect.StringKind:
					if !utf8.ValidString(list.Get(i).String()) {
						count++
					}
				case protoreflect.MessageKind:
					count += countInvalidStringFields(list.Get(i).Message())
				default:
				}
			}
		case fd.Kind() == protoreflect.StringKind:
			if !utf8.ValidString(v.String()) {
				count++
			}
		case fd.Kind() == protoreflect.MessageKind:
			count += countInvalidStringFields(v.Message())
		default:
		}
		return true
	})
	return count
}

// TestSanitizeUTF8MessageCoversEveryStringField proves the BYT-9916 fix
// covers every string field of the database metadata — names, types,
// defaults, definitions — not just comments: fill each one with the raw-GBK
// shape go-ora leaks for values ending in a dangling lead byte, then assert
// sanitization leaves zero invalid strings and proto marshaling succeeds.
func TestSanitizeUTF8MessageCoversEveryStringField(t *testing.T) {
	// 测试 + dangling lead byte 0xb1: the exact wholly-unconverted raw GBK
	// shape go-ora returns (engine-verified against Oracle 11gR2/ZHS16GBK).
	rawGBK := "\xb2\xe2\xca\xd4\xb1"
	require.False(t, utf8.ValidString(rawGBK))

	metadata := &storepb.DatabaseSchemaMetadata{}
	fillAllStringFields(metadata.ProtoReflect(), rawGBK, 8)

	invalidBefore := countInvalidStringFields(metadata.ProtoReflect())
	require.Greater(t, invalidBefore, 100,
		"filler must reach the full schema shape; got only %d string fields", invalidBefore)
	_, err := proto.Marshal(metadata)
	require.Error(t, err, "pre-sanitize marshal must fail — otherwise this test proves nothing")

	common.SanitizeUTF8Message(metadata)

	require.Equal(t, 0, countInvalidStringFields(metadata.ProtoReflect()),
		"sanitization must leave zero invalid string values anywhere in the message")
	_, err = proto.Marshal(metadata)
	require.NoError(t, err)
	_, err = protojson.Marshal(metadata)
	require.NoError(t, err)
}

// TestSanitizeUTF8MessageNameFields pins the customer-visible BYT-9916 shape:
// object names (schema/table/column) carrying raw GBK bytes must marshal
// after sanitization, and valid names must pass through untouched.
func TestSanitizeUTF8MessageNameFields(t *testing.T) {
	rawGBK := "AB\xe6"
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: rawGBK,
		Schemas: []*storepb.SchemaMetadata{{
			Name: rawGBK,
			Tables: []*storepb.TableMetadata{{
				Name: rawGBK,
				Columns: []*storepb.ColumnMetadata{
					{Name: rawGBK, Type: rawGBK, Default: rawGBK},
					{Name: "测试列", Type: "VARCHAR2(50)"},
				},
			}},
		}},
	}
	_, err := proto.Marshal(metadata)
	require.Error(t, err)

	common.SanitizeUTF8Message(metadata)
	_, err = proto.Marshal(metadata)
	require.NoError(t, err)
	require.Equal(t, "测试列", metadata.Schemas[0].Tables[0].Columns[1].Name,
		"valid UTF-8 must pass through unchanged")
	require.True(t, utf8.ValidString(metadata.Schemas[0].Tables[0].Name))
}
