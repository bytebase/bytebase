package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestEventTriggerSDLSingleFileOutput(t *testing.T) {
	metadata := eventTriggerSDLMetadata()

	result, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, metadata)
	require.NoError(t, err)
	assert.Contains(t, result, `CREATE FUNCTION "public"."audit_ddl"() RETURNS event_trigger`)
	assert.Contains(t, result, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, result, `WHEN TAG IN ('CREATE TABLE')`)
	assert.Contains(t, result, `EXECUTE FUNCTION "public"."audit_ddl"();`)
	assert.Contains(t, result, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)

	sdl, err := schema.MetadataToSDL(
		storepb.Engine_POSTGRES,
		model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true),
	)
	require.NoError(t, err)
	assert.Contains(t, sdl, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, sdl, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)
}

func TestEventTriggerSDLMultiFileOutput(t *testing.T) {
	result, err := GetMultiFileDatabaseDefinition(
		schema.GetDefinitionContext{SDLFormat: true},
		eventTriggerSDLMetadata(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	fileMap := make(map[string]string)
	for _, file := range result.Files {
		fileMap[file.Name] = file.Content
	}

	eventTriggerFile, ok := fileMap["event_triggers.sql"]
	require.True(t, ok, "event_triggers.sql file should exist")
	assert.Contains(t, eventTriggerFile, `CREATE EVENT TRIGGER "audit_ddl_start" ON ddl_command_start`)
	assert.Contains(t, eventTriggerFile, `WHEN TAG IN ('CREATE TABLE')`)
	assert.Contains(t, eventTriggerFile, `EXECUTE FUNCTION "public"."audit_ddl"();`)
	assert.Contains(t, eventTriggerFile, `COMMENT ON EVENT TRIGGER "audit_ddl_start" IS 'Audit DDL start';`)
}

func eventTriggerSDLMetadata() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Functions: []*storepb.FunctionMetadata{
					{
						Name:      "audit_ddl",
						Signature: "audit_ddl()",
						Definition: `CREATE FUNCTION "public"."audit_ddl"() RETURNS event_trigger
LANGUAGE plpgsql
AS $$
BEGIN
END;
$$`,
					},
				},
			},
		},
		EventTriggers: []*storepb.EventTriggerMetadata{
			{
				Name:           "audit_ddl_start",
				Event:          "ddl_command_start",
				Tags:           []string{"CREATE TABLE"},
				FunctionSchema: "public",
				FunctionName:   "audit_ddl",
				Enabled:        true,
				Comment:        "Audit DDL start",
			},
		},
	}
}
