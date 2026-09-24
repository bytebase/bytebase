package tests

// An MCP session applies the caller's own masking provisioning exactly as the
// console does. Two mechanisms let a user see a real value, and they reach
// masking through different doors: a masking exemption is filtered per caller
// inside the masking pass (QueryResultMasker.exemptionsForPrincipal), while an
// access grant's unmask turns the pass off at the query context
// (db.QueryContext.SkipMasking), which Query and Export each set on their own.
// Every read below asserts both transports on one fixture, so an agent that
// reads something other than what the console reads fails in either direction.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	// maskPlaceholder is what a full mask puts in place of a value when the
	// admin chose no substitution, and it is the whole of the masked-write
	// guard's detectable set (masker.DefaultFullMaskSubstitution). The literal
	// is written out rather than imported from the masker package: it is what
	// an agent actually reads back and what it must not write, so a test that
	// borrowed the constant would follow a drift instead of catching it.
	maskPlaceholder = "******"
	// maskedSecret is the value behind the mask, distinctive enough that an
	// assertion can say the real data reached the caller rather than only that
	// the placeholder did not.
	maskedSecret = "hunter2-in-the-clear"
	// secretQuery is the one statement every read here runs. An access grant
	// matches on the exact query text, so a second spelling of the same read
	// would need a second grant.
	secretQuery = "SELECT secret FROM employee"
)

// mcpMaskingFixture is one live server holding one Postgres row whose `secret`
// column carries a full-mask semantic type, reachable two ways: an open MCP
// session and the console clients on ctl. Both belong to the same principal,
// which is what makes every assertion below about the transport rather than
// about who is asking.
type mcpMaskingFixture struct {
	ctl *controller
	ctx context.Context
	// database is the full resource name, for the API; name is the short one
	// query_database resolves on.
	database    string
	name        string
	workspaceID string
	session     *mcp.ClientSession
}

func setupMCPMaskingFixture(ctx context.Context, t *testing.T) *mcpMaskingFixture {
	t.Helper()
	a := require.New(t)
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)

	// The substitution is spelled out rather than left to the default, because
	// the masked-write guard recognizes this one string: a workspace that
	// configured any other substitution is outside what the guard can detect,
	// and pinning the recognizable one is what makes the write half testable.
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_SEMANTIC_TYPES.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_SemanticType{
					SemanticType: &v1pb.SemanticTypeSetting{
						Types: []*v1pb.SemanticTypeSetting_SemanticType{
							{
								Id:    "default",
								Title: "Default",
								Algorithm: &v1pb.Algorithm{
									Mask: &v1pb.Algorithm_FullMask_{
										FullMask: &v1pb.Algorithm_FullMask{Substitution: maskPlaceholder},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	container := sharedPgTarget(t)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("mcp-masking"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "MCP masking",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{container.adminDataSource()},
		},
	}))
	a.NoError(err)

	databaseName := generateRandomString("maskdb")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, databaseName, ""))
	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, databaseName),
	}))
	a.NoError(err)

	setup, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{Content: fmt.Appendf(nil,
			`CREATE TABLE employee(id INT PRIMARY KEY, name TEXT, secret TEXT); INSERT INTO employee VALUES (1, 'Bytebase', '%s');`,
			maskedSecret)},
	}))
	a.NoError(err)
	a.NoError(ctl.changeDatabase(ctx, ctl.project, databaseResp.Msg, setup.Msg, false))

	// Postgres puts the table in `public`; the catalog is keyed by the schema
	// the query span resolves, so an empty schema name here would attach the
	// semantic type to nothing and every mask assertion below would be vacuous.
	_, err = ctl.databaseCatalogServiceClient.UpdateDatabaseCatalog(ctx, connect.NewRequest(&v1pb.UpdateDatabaseCatalogRequest{
		Catalog: &v1pb.DatabaseCatalog{
			Name: fmt.Sprintf("%s/catalog", databaseResp.Msg.Name),
			Schemas: []*v1pb.SchemaCatalog{
				{
					Name: "public",
					Tables: []*v1pb.TableCatalog{
						{
							Name: "employee",
							Kind: &v1pb.TableCatalog_Columns_{
								Columns: &v1pb.TableCatalog_Columns{
									Columns: []*v1pb.ColumnCatalog{
										{Name: "secret", SemanticType: "default"},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	a.NoError(err)

	// READ_WRITE rather than READ_ONLY: Export and the masked-write guard's
	// doors sit on methods a read-only ceiling already refuses for a different
	// reason, and a read is served under either ceiling.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	t.Cleanup(func() { session.Close() })

	f := &mcpMaskingFixture{
		ctl:         ctl,
		ctx:         ctx,
		database:    databaseResp.Msg.Name,
		name:        databaseName,
		workspaceID: strings.TrimPrefix(workspace.Msg.Name, "workspaces/"),
		session:     session,
	}

	// The precondition every later assertion rests on. With nothing granted to
	// anybody the column is masked on both paths, so a real value read later is
	// the provisioning at work rather than a semantic type that never attached.
	a.Equal(maskPlaceholder, f.consoleReadsSecret(t),
		"precondition: the semantic type must actually mask the column")
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, session),
		"precondition: and mask it over MCP too")
	return f
}

// consoleReadsSecret is the human path: a person signed in to Bytebase running
// the same statement the agent runs.
func (f *mcpMaskingFixture) consoleReadsSecret(t *testing.T) string {
	t.Helper()
	resp, err := f.ctl.sqlServiceClient.Query(f.ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      f.database,
		Statement: secretQuery,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 1)
	require.Empty(t, resp.Msg.Results[0].Error)
	require.Len(t, resp.Msg.Results[0].Rows, 1)
	require.Len(t, resp.Msg.Results[0].Rows[0].Values, 1)
	return resp.Msg.Results[0].Rows[0].Values[0].GetStringValue()
}

// agentReadsSecret is the same read through query_database, the tool an agent
// actually reaches for.
func (f *mcpMaskingFixture) agentReadsSecret(t *testing.T, session *mcp.ClientSession) string {
	t.Helper()
	read := queryDatabaseOnSession(f.ctx, t, session, f.name, secretQuery)
	require.False(t, read.isError, "the agent's read must be served, or nothing was masked: %s", read.text)
	require.Len(t, read.output.Rows, 1)
	require.Len(t, read.output.Rows[0], 1)
	value, ok := read.output.Rows[0][0].(string)
	require.True(t, ok, "a TEXT column reaches the agent as a string, masked or not: %v", read.output.Rows[0][0])
	return value
}

// exemptFromMasking grants member a masking exemption over the whole project
// and returns the policy's name. An empty condition means every database with
// no expiration, which keeps the exemption's CEL out of what is under test.
func (f *mcpMaskingFixture) exemptFromMasking(t *testing.T, member string) string {
	t.Helper()
	policy, err := f.ctl.orgPolicyServiceClient.CreatePolicy(f.ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: f.ctl.project.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_MASKING_EXEMPTION,
			Policy: &v1pb.Policy_MaskingExemptionPolicy{
				MaskingExemptionPolicy: &v1pb.MaskingExemptionPolicy{
					Exemptions: []*v1pb.MaskingExemptionPolicy_Exemption{
						{Members: []string{member}, Condition: &expr.Expr{}},
					},
				},
			},
		},
	}))
	require.NoError(t, err)
	return policy.Msg.Name
}

// grantUnmask creates and activates an access grant that unmasks exactly
// secretQuery on the fixture's database.
//
// It reaches masking through a different door from an exemption: it turns the
// whole masking pass off at the query context (db.QueryContext.SkipMasking)
// rather than filtering the caller's exemptions inside it.
//
// preCheckAccess finds a grant only on an exact match of creator, trimmed query
// text, ACTIVE status and a live expire_time, so all four are pinned here
// instead of being left to the issue workflow that also activates grants.
func (f *mcpMaskingFixture) grantUnmask(t *testing.T, export bool) {
	t.Helper()
	a := require.New(t)

	// preCheckAccess returns early unless the project allows JIT access, so
	// without this the grant would exist and never be consulted.
	_, err := f.ctl.projectServiceClient.UpdateProject(f.ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project:    &v1pb.Project{Name: f.ctl.project.Name, AllowJustInTimeAccess: true},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_just_in_time_access"}},
	}))
	a.NoError(err)

	created, err := f.ctl.accessGrantServiceClient.CreateAccessGrant(f.ctx, connect.NewRequest(&v1pb.CreateAccessGrantRequest{
		Parent: f.ctl.project.Name,
		AccessGrant: &v1pb.AccessGrant{
			Creator:    f.ctl.principalName,
			Targets:    []string{f.database},
			Query:      secretQuery,
			Reason:     "MCP masking end-to-end coverage",
			Unmask:     true,
			Export:     export,
			Expiration: &v1pb.AccessGrant_ExpireTime{ExpireTime: timestamppb.New(time.Now().Add(time.Hour))},
		},
	}))
	a.NoError(err)

	active, err := f.ctl.accessGrantServiceClient.ActivateAccessGrant(f.ctx, connect.NewRequest(&v1pb.ActivateAccessGrantRequest{
		Name: created.Msg.Name,
	}))
	a.NoError(err)
	a.Equal(v1pb.AccessGrant_ACTIVE, active.Msg.Status)
	a.True(active.Msg.GetExpireTime().AsTime().After(time.Now()),
		"the grant has to be live, or preCheckAccess filters it out and the test proves nothing")
}

// storeRetiredMaskingKey writes the retired ignoreMaskingExemptions key into
// the MCP setting row, set to true, as a workspace that turned the toggle on
// before 3.23.0 still holds it. Direct SQL is the only way to write it: the
// settings API no longer has the field.
func (f *mcpMaskingFixture) storeRetiredMaskingKey(t *testing.T) {
	t.Helper()
	a := require.New(t)
	db, err := sql.Open("pgx", f.ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	result, err := db.ExecContext(f.ctx, `
		UPDATE setting SET value = value || '{"ignoreMaskingExemptions": true}'::jsonb
		WHERE workspace = $1 AND name = 'MCP';
	`, f.workspaceID)
	a.NoError(err)
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the MCP setting row must exist for the retired key to mean anything")
}

// agentExportsCSV runs SQLService/Export over an MCP session. No MCP tool wraps
// Export, so call_api is the shape an agent actually reaches it in.
func (f *mcpMaskingFixture) agentExportsCSV(t *testing.T, session *mcp.ClientSession) string {
	t.Helper()
	out := callAPIOnSession(f.ctx, t, session, "SQLService/Export", map[string]any{
		"name":      f.database,
		"statement": secretQuery,
		"format":    "CSV",
	})
	require.Equal(t, http.StatusOK, out.Status, "the export must be served: %s", out.Error)
	var payload struct {
		Content []byte `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(out.RawResponse), &payload))
	return exportedCSV(t, payload.Content)
}

func (f *mcpMaskingFixture) consoleExportsCSV(t *testing.T) string {
	t.Helper()
	resp, err := f.ctl.sqlServiceClient.Export(f.ctx, connect.NewRequest(&v1pb.ExportRequest{
		Name:      f.database,
		Statement: secretQuery,
		Format:    v1pb.ExportFormat_CSV,
	}))
	require.NoError(t, err)
	return exportedCSV(t, resp.Msg.Content)
}

// exportedCSV returns the result entries of an export payload. An export is a
// zip carrying a .sql statement file and a .result.<format> file per statement,
// so the statement half is skipped: it echoes the SQL the caller sent and would
// match a value assertion for a reason that has nothing to do with masking.
func exportedCSV(t *testing.T, content []byte) string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	var sb strings.Builder
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".result.csv") {
			continue
		}
		sb.WriteString(readZipEntry(t, file))
	}
	require.NotEmpty(t, sb.String(), "the export must carry a CSV result entry")
	return sb.String()
}

func readZipEntry(t *testing.T, file *zip.File) string {
	t.Helper()
	entry, err := file.Open()
	require.NoError(t, err)
	defer entry.Close()
	body, err := io.ReadAll(entry)
	require.NoError(t, err)
	return string(body)
}

// changeToolResult is what an agent gets back from propose_database_change:
// the text a model reads and whether the tool reported a failure.
type changeToolResult struct {
	text    string
	isError bool
}

func proposeChangeOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, database, statement, title string) changeToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "propose_database_change",
		Arguments: map[string]any{
			"database": database,
			"sql":      statement,
			"title":    title,
		},
	})
	require.NoError(t, err)
	out := changeToolResult{isError: result.IsError}
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	out.text = sb.String()
	return out
}

// sheetBody is the CreateSheet request an agent sends through call_api. Sheet
// content is bytes, so it crosses as base64 — which is also why the guard scans
// the decoded statement rather than the wire text.
func sheetBody(project, statement string) map[string]any {
	return map[string]any{
		"parent": project,
		"sheet":  map[string]any{"content": base64.StdEncoding.EncodeToString([]byte(statement))},
	}
}

// TestMCPMaskingFollowsTheCallersProvisioning pins console parity on every
// path that serves a real value: a masking exemption, an access grant's unmask
// on Query, and the same unmask on Export, which sets SkipMasking on its own.
//
// The row carries the retired ignoreMaskingExemptions key set to true. The
// store discards a key it does not define, so the key must change nothing.
func TestMCPMaskingFollowsTheCallersProvisioning(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)
	f.storeRetiredMaskingKey(t)

	exemption := f.exemptFromMasking(t, "user:demo@example.com")
	a.Equal(maskedSecret, f.consoleReadsSecret(t),
		"the exemption serves the real value in the console")
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"and to the same caller's MCP session")

	// While the exemption is live every read is real, so the grant is checked
	// with it gone.
	_, err := f.ctl.orgPolicyServiceClient.DeletePolicy(f.ctx, connect.NewRequest(&v1pb.DeletePolicyRequest{
		Name: exemption,
	}))
	a.NoError(err)
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, f.session),
		"precondition: without the exemption the column masks again")

	f.grantUnmask(t, true /* export */)
	a.Equal(maskedSecret, f.consoleReadsSecret(t), "the grant unmasks in the console")
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"and the same grant unmasks the caller's MCP session")
	a.Contains(f.consoleExportsCSV(t), maskedSecret,
		"the grant exports in the clear from the console")
	a.Contains(f.agentExportsCSV(t, f.session), maskedSecret,
		"and Export over MCP honors the same grant")
}

// TestMCPMaskedWriteIsRefused is the corruption the read half would otherwise
// invite: the agent reads a masked column, gets the placeholder, and writes the
// placeholder back as if it were the value. The real data is gone and nothing
// in the change looks wrong.
//
// Every door change SQL reaches the pipeline through is probed, because one
// unguarded door is the whole hole: CreateSheet is where propose_database_change
// and a hand-rolled plan both land, BatchCreateSheets is the same door in bulk,
// CreateRelease turns inline file statements into sheets without SheetService
// being called at all, and Query and Export need no proposal at all: Query is
// READ class, so every ceiling that opens a session serves it, and both
// authorize a write under this fixture's READ_WRITE ceiling.
//
// A clean statement through the same door at the end is what proves the door
// is genuinely open — otherwise "refused" and "not reachable" look alike.
//
// The RED state: remove the masked-write entries from mcpRequestShapeRefusals
// and no refusal here is the guard's any more — the sheet, release and query
// rows answer 200, and the export row answers 500, because validateQueryRequest
// returns a bare *queryError that carries no connect code. Which is why each
// assertion names the placeholder and not only the status.
func TestMCPMaskedWriteIsRefused(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	// The console needs to see the real value, or "the write never ran" is
	// unprovable: a masked read shows the placeholder whether or not the UPDATE
	// landed.
	f.exemptFromMasking(t, "user:demo@example.com")
	a.Equal(maskedSecret, f.consoleReadsSecret(t))

	maskedWrite := fmt.Sprintf("UPDATE employee SET secret = '%s' WHERE id = 1", maskPlaceholder)
	const cleanWrite = "UPDATE employee SET secret = 'rotated-by-hand' WHERE id = 1"

	refused := callAPIOnSession(f.ctx, t, f.session, "SheetService/CreateSheet",
		sheetBody(f.ctl.project.Name, maskedWrite))
	a.Equal(http.StatusForbidden, refused.Status,
		"writing the placeholder back must be refused: %s", refused.Error)
	a.Contains(refused.Error, maskPlaceholder,
		"the refusal must name the literal, or the agent cannot tell what to remove")

	served := callAPIOnSession(f.ctx, t, f.session, "SheetService/CreateSheet",
		sheetBody(f.ctl.project.Name, cleanWrite))
	a.Equal(http.StatusOK, served.Status,
		"an ordinary change is still served: %s", served.Error)

	// The batch door refuses on one offending sheet, matching the clamp, which
	// denies a batch holding any statement it will not serve.
	batch := callAPIOnSession(f.ctx, t, f.session, "SheetService/BatchCreateSheets", map[string]any{
		"parent": f.ctl.project.Name,
		"requests": []any{
			sheetBody(f.ctl.project.Name, cleanWrite),
			sheetBody(f.ctl.project.Name, maskedWrite),
		},
	})
	a.Equal(http.StatusForbidden, batch.Status,
		"one masked sheet must refuse the whole batch: %s", batch.Error)
	a.Contains(batch.Error, maskPlaceholder)

	// The execution door. No sheet, no plan, no rollout: the statement the agent
	// just composed from a masked read goes straight at the table.
	direct := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": maskedWrite,
	})
	a.Equal(http.StatusForbidden, direct.Status,
		"a masked write must be refused on the execution door too: %s", direct.Error)
	a.Contains(direct.Error, maskPlaceholder)
	a.Equal(maskedSecret, f.consoleReadsSecret(t),
		"and the refusal must land before the statement runs")

	// Export takes a raw statement too, and on MySQL its handler skips the
	// read-only validation — which is why it is a write door there. On this
	// Postgres fixture the validation is what would refuse it instead, so the
	// assertion below is on the message rather than on the status alone.
	exported := callAPIOnSession(f.ctx, t, f.session, "SQLService/Export", map[string]any{
		"name":      f.database,
		"statement": maskedWrite,
		"format":    "CSV",
	})
	a.Equal(http.StatusForbidden, exported.Status,
		"the export door carries the same statement: %s", exported.Error)
	a.Contains(exported.Error, maskPlaceholder)

	release := callAPIOnSession(f.ctx, t, f.session, "ReleaseService/CreateRelease", map[string]any{
		"parent": f.ctl.project.Name,
		"release": map[string]any{
			"type": "VERSIONED",
			"files": []any{
				map[string]any{
					"path":      "1.0/V0001_rotate.sql",
					"version":   "0001",
					"statement": base64.StdEncoding.EncodeToString([]byte(maskedWrite)),
				},
			},
		},
	})
	a.Equal(http.StatusForbidden, release.Status,
		"a release carrying the placeholder inline must be refused too: %s", release.Error)
	a.Contains(release.Error, maskPlaceholder,
		"the refusal must be the mask guard's, not some other precondition of CreateRelease")

	// The tool an agent actually reaches for stops at its first step, sheet
	// creation, so nothing downstream — plan, issue, rollout — is left behind.
	// It has to carry the refusal's own wording: a coaching message the primary
	// tool rewrites into generic permission advice would send the agent at a
	// permission it already holds (checkAPIResponse, tool_change.go).
	change := proposeChangeOnSession(f.ctx, t, f.session, f.name, maskedWrite, "Rotate the secret")
	a.True(change.isError, "propose_database_change must fail rather than build a plan: %s", change.text)
	a.Contains(change.text, maskPlaceholder,
		"the tool an agent actually calls must name the literal too")

	// A person in the console writes whatever they like: the guard lives on the
	// MCP chain, and a human who reads the real value has no placeholder to
	// write back by accident.
	created, err := f.ctl.sheetServiceClient.CreateSheet(f.ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: f.ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte(maskedWrite)},
	}))
	a.NoError(err, "the console path must be untouched by the mask guard")
	a.NotEmpty(created.Msg.Name)

	// And the door really is open, which "refused" alone cannot show. A clean
	// write through SQLService/Query is the only place in this PR an MCP session
	// runs DML end to end, so it is asserted on the stored value, not the status.
	const agentWrite = "UPDATE employee SET secret = 'rotated-by-agent' WHERE id = 1"
	agentRan := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": agentWrite,
	})
	a.Equal(http.StatusOK, agentRan.Status, "the execution door is open: %s", agentRan.Error)
	a.Equal("rotated-by-agent", f.consoleReadsSecret(t),
		"the agent's write landed, so the refusals above were the guard and not a closed door")
}
