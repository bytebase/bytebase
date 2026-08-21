package tests

// ignore_masking_exemptions is the workspace toggle these tests exercise: an
// MCP request stops applying the caller's own unmasking provisioning.
//
// Two separate mechanisms let a user see a real value, and the toggle has to
// suppress both, so each gets a test of its own here: the masking exemptions
// granted to them (TestMCPMaskingIgnoresTheUsersExemption) and the unmask
// carried by an access grant (TestMCPMaskingIgnoresTheAccessGrantUnmask).
// Suppressing only one would leave the other serving real values to the agent,
// and no unit test composes both — the exemption seam and SkipMasking sit on
// different sides of the masking pass.
//
// It is not a confidentiality boundary. Masking runs only where the engine
// supports it and only on classified columns, so this narrows what an agent
// reads through the paths Bytebase masks and nothing else.
//
// The other half of the claim is the console: the toggle keys on the delegated
// grant, so the same user reading the same column signed in to Bytebase is
// untouched. Every read test below asserts both paths on one fixture, because
// a mask that appeared on both would satisfy "the agent sees a mask" while
// destroying the feature.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// setIgnoreMaskingExemptions writes the workspace masking toggle through the
// settings API. There is no console control yet — the toggle card is 1b-6 — so
// this is the only way to set it today.
func (ctl *controller) setIgnoreMaskingExemptions(ctx context.Context, ignore bool) error {
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Mcp{
					Mcp: &v1pb.MCPSetting{IgnoreMaskingExemptions: ignore},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.mcp.ignore_masking_exemptions"},
		},
	}))
	return err
}

func (ctl *controller) getIgnoreMaskingExemptions(ctx context.Context) (bool, error) {
	resp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/" + v1pb.Setting_MCP.String(),
	}))
	if err != nil {
		return false, err
	}
	return resp.Msg.Value.GetMcp().GetIgnoreMaskingExemptions(), nil
}

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
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	t.Cleanup(func() { ctl.Close(ctx) })

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

	container, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("mcp-masking"),
		Instance: &v1pb.Instance{
			Title:       "MCP masking",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{container.adminDataSource()},
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

	// READ_WRITE rather than READ_ONLY: the toggle's write half — the
	// masked-write guard — sits on methods a read-only ceiling already refuses
	// for a different reason, and a read is served under either ceiling.
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
	// anybody the column is masked on both paths, so a test that expects a mask
	// is testing the toggle rather than a fixture where the semantic type never
	// attached.
	a.Equal(maskPlaceholder, f.consoleReadsSecret(t),
		"precondition: the semantic type must actually mask the column")
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, session),
		"precondition: and mask it over MCP too")
	return f
}

// consoleReadsSecret is the human path: a person signed in to Bytebase running
// the same statement the agent runs. It carries no delegated grant, which is
// the only thing the toggle keys on.
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

// exemptFromMasking grants member a masking exemption over the whole project.
// An empty condition means every database with no expiration, which keeps the
// exemption out of the way of what is under test: whether the toggle suppresses
// it, not whether its CEL matched.
func (f *mcpMaskingFixture) exemptFromMasking(t *testing.T, member string) {
	t.Helper()
	_, err := f.ctl.orgPolicyServiceClient.CreatePolicy(f.ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
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
}

// grantUnmask creates and activates an access grant that unmasks exactly
// secretQuery on the fixture's database.
//
// This is the second mechanism the toggle suppresses, and it reaches masking
// through a different door from an exemption: it turns the whole masking pass
// off at the query context (db.QueryContext.SkipMasking) rather than filtering
// the caller's exemptions inside it.
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
			Reason:     "masking toggle end-to-end coverage",
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

// storedMaskingToggleKey reports whether the workspace profile row carries a
// ignoreMaskingExemptions key at all.
//
// protojson omits a false bool, so "never set" and "set to false" are the same
// row. That is exactly why the resolver has to read the missing key as false
// rather than as unknown, and why the absent case is worth pinning separately
// from the explicit one.
func (f *mcpMaskingFixture) storedMaskingToggleKey(t *testing.T) bool {
	t.Helper()
	db, err := sql.Open("pgx", f.ctl.profile.PgURL)
	require.NoError(t, err)
	defer db.Close()
	var present bool
	require.NoError(t, db.QueryRowContext(f.ctx, `
		SELECT jsonb_exists(value, 'ignoreMaskingExemptions') FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, f.workspaceID).Scan(&present))
	return present
}

// writeMaskingToggleOutOfBand flips the toggle behind the server's back. The
// setting cache has no TTL and only in-process writes refresh it, so this is
// the one state a cached read would miss — a direct SQL edit, or another
// replica's write.
func (f *mcpMaskingFixture) writeMaskingToggleOutOfBand(t *testing.T, ignore bool) {
	t.Helper()
	a := require.New(t)
	db, err := sql.Open("pgx", f.ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	result, err := db.ExecContext(f.ctx, `
		UPDATE setting SET value = jsonb_set(value, '{ignoreMaskingExemptions}', $2::jsonb)
		WHERE workspace = $1 AND name = 'MCP';
	`, f.workspaceID, strconv.FormatBool(ignore))
	a.NoError(err)
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the MCP setting row must exist for the out-of-band write to mean anything")
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

// queryValueOnSession runs SQLService/Query through call_api and returns the
// single value the statement selects.
//
// call_api rather than query_database wherever the session belongs to a
// non-admin: query_database resolves the database by listing workspace-wide
// first, so a listing denial would refuse the read for a reason that is not the
// one under test.
func queryValueOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, database, statement string) string {
	t.Helper()
	a := require.New(t)
	out := callAPIOnSession(ctx, t, session, "SQLService/Query", map[string]any{
		"name":      database,
		"statement": statement,
	})
	a.Equal(http.StatusOK, out.Status, "the read must be served: %s", out.Error)
	var payload struct {
		Results []struct {
			Error string `json:"error"`
			Rows  []struct {
				Values []struct {
					StringValue string `json:"stringValue"`
				} `json:"values"`
			} `json:"rows"`
		} `json:"results"`
	}
	a.NoError(json.Unmarshal([]byte(out.RawResponse), &payload))
	a.Len(payload.Results, 1)
	a.Empty(payload.Results[0].Error)
	a.Len(payload.Results[0].Rows, 1)
	a.Len(payload.Results[0].Rows[0].Values, 1)
	return payload.Results[0].Rows[0].Values[0].StringValue
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

// TestMCPMaskingIgnoresTheUsersExemption is the headline: the toggle
// suppresses the caller's OWN masking provisioning, not merely the masking they
// were never granted an exemption from.
//
// One principal, one column, two transports. The exemption is live throughout —
// the control at the top proves it by serving the real value to both paths with
// the toggle off — so the only thing that changes between the two halves is
// which transport the request arrived on.
//
// The RED state: drop the mcpIgnoresMaskingExemptions check from
// QueryResultMasker.exemptionsForPrincipal and the agent's read goes back to
// the real value, because the exemption is filtered to this caller and passes.
func TestMCPMaskingIgnoresTheUsersExemption(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	f.exemptFromMasking(t, "user:demo@example.com")

	// With the toggle off the exemption reaches both paths, so it is the
	// toggle and not the exemption that the second half turns on.
	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, false))
	a.Equal(maskedSecret, f.consoleReadsSecret(t),
		"the exemption must serve the real value in the console")
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"and with the toggle off the agent inherits the same exemption")

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))

	a.Equal(maskedSecret, f.consoleReadsSecret(t),
		"the toggle is about what an agent may read; the console keeps the exemption")
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, f.session),
		"the toggle must stop applying the exemption the caller holds")
}

// TestMCPMaskingIgnoresTheAccessGrantUnmask is the other mechanism.
// An access grant with unmask=true never reaches the exemption filter at all:
// it sets db.QueryContext.SkipMasking, which turns the whole masking pass off
// one layer earlier. So suppressing exemptions alone would leave this path
// handing the agent real values.
//
// The RED state: drop `&& !mcpIgnoresMaskingExemptions(ctx)` from the SkipMasking term in
// SQLService.Query and the agent's read under the toggle goes back to the real
// value.
func TestMCPMaskingIgnoresTheAccessGrantUnmask(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	f.grantUnmask(t, false /* export */)

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, false))
	a.Equal(maskedSecret, f.consoleReadsSecret(t), "the grant unmasks in the console")
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"and with the toggle off the same grant unmasks over MCP")

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))

	a.Equal(maskedSecret, f.consoleReadsSecret(t),
		"the human's grant is untouched — the toggle did not revoke it")
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, f.session),
		"the toggle must stop applying the unmask an access grant carries")
}

// TestMCPMaskingIgnoresExemptionsOnExport covers the second way data leaves the
// product. Export is a WRITE method, so a READ_WRITE MCP session reaches it,
// and it carries its own copy of the access-grant unmask rather than sharing
// Query's — one enforcement point per copy, which is why suppressing it in
// Query alone would leave the whole dataset exportable in the clear.
//
// The RED state: drop `&& !mcpIgnoresMaskingExemptions(ctx)` from the skipMasking term in
// SQLService.Export and the CSV under the toggle carries the real value.
func TestMCPMaskingIgnoresExemptionsOnExport(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	f.grantUnmask(t, true /* export */)

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, false))
	a.Contains(f.agentExportsCSV(t, f.session), maskedSecret,
		"with the toggle off the grant unmasks the agent's export too")

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))

	exported := f.agentExportsCSV(t, f.session)
	a.Contains(exported, maskPlaceholder, "an export under the toggle must carry the mask")
	a.NotContains(exported, maskedSecret, "and must not carry the value behind it")

	a.Contains(f.consoleExportsCSV(t), maskedSecret,
		"the same grant still exports in the clear for a person in the console")
}

// TestMCPMaskingToggleOffFollowsProvisioning pins the two states an existing workspace
// can be in. Absent is the one that matters on upgrade: nothing wrote the key,
// and a resolver that read the missing key as anything but false would start
// masking for every workspace that upgraded into this build.
//
// Explicit false is stored the same way — protojson omits a false bool — so the
// row is identical and the assertion is that the API round trip and the served
// behavior agree with it.
func TestMCPMaskingToggleOffFollowsProvisioning(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	f.exemptFromMasking(t, "user:demo@example.com")

	a.False(f.storedMaskingToggleKey(t),
		"precondition: the fixture wrote a ceiling and nothing else, so the toggle was never set")
	stored, err := f.ctl.getIgnoreMaskingExemptions(f.ctx)
	a.NoError(err)
	a.False(stored)
	a.Equal(maskedSecret, f.consoleReadsSecret(t))
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"a workspace that never set the toggle follows each user's own provisioning")

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, false))

	a.False(f.storedMaskingToggleKey(t),
		"a false bool is stored as absence, which is why both states must resolve the same way")
	stored, err = f.ctl.getIgnoreMaskingExemptions(f.ctx)
	a.NoError(err)
	a.False(stored)
	a.Equal(maskedSecret, f.consoleReadsSecret(t))
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"an explicit false must behave exactly as absent does")
}

// TestMCPMaskingToggleSparesHumanSessions is the half of the claim that
// keeps the toggle shippable. It keys on the delegated grant a session carries,
// never on the user or on a field value, so a person signed in to the console
// is untouched whatever the workspace tells agents to ignore.
//
// The principal here is a non-admin who holds the exemption, so the console
// half cannot pass by way of some admin bypass, and the MCP half runs on the
// SAME principal to show the toggle really is on while the console read serves
// the real value.
//
// The RED state: key mcpIgnoresMaskingExemptions on anything other than the grant's
// presence — the consented scope, say, which a legacy session leaves empty —
// and either the console read starts masking or the MCP read stops.
func TestMCPMaskingToggleSparesHumanSessions(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	const readerEmail = "masking-reader@example.com"
	const readerPassword = "1024bytebase"
	reader, err := f.ctl.userServiceClient.CreateUser(f.ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Title: "masking reader", Email: readerEmail, Password: readerPassword},
	}))
	a.NoError(err)
	_, err = f.ctl.addMemberToWorkspaceIAM(f.ctx, reader.Msg.Workspace, "user:"+readerEmail, "roles/workspaceMember")
	a.NoError(err)

	policy, err := f.ctl.projectServiceClient.GetIamPolicy(f.ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: f.ctl.project.Name,
	}))
	a.NoError(err)
	policy.Msg.Bindings = append(policy.Msg.Bindings, &v1pb.Binding{
		Role: "roles/sqlEditorUser", Members: []string{"user:" + readerEmail},
	})
	_, err = f.ctl.projectServiceClient.SetIamPolicy(f.ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: f.ctl.project.Name,
		Policy:   policy.Msg,
	}))
	a.NoError(err)

	f.exemptFromMasking(t, "user:"+readerEmail)
	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))

	login, err := f.ctl.authServiceClient.Login(f.ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email: readerEmail, Password: readerPassword,
	}))
	a.NoError(err)

	admin := f.ctl.authInterceptor.token
	f.ctl.authInterceptor.token = login.Msg.Token
	console := f.consoleReadsSecret(t)
	f.ctl.authInterceptor.token = admin
	a.Equal(maskedSecret, console,
		"a person signed in to Bytebase keeps their exemption while agents lose it")

	// The same principal one transport over, so the toggle is demonstrably on
	// rather than merely written.
	readerMCP, _ := mintMCPOAuthToken(t, f.ctl, login.Msg.Token)
	session := openMCPSession(f.ctx, t, f.ctl, readerMCP)
	defer session.Close()
	a.Equal(maskPlaceholder, queryValueOnSession(f.ctx, t, session, f.database, secretQuery),
		"the same user's agent session is held to the mask")
}

// TestMCPMaskingToggleBitesTheNextRequest is the live-state pin. The gate reads
// the MCP setting straight from the database on every request and stamps
// what it read, so an admin flipping the toggle binds the next request of a
// session that is already open: no re-consent, no reconnect, no restart.
//
// The out-of-band half is what makes the claim about the READ rather than about
// the write. An in-process UpdateSetting refreshes the setting cache, so a
// cached read would satisfy it too; a direct SQL edit is the state only an
// uncached read answers correctly.
//
// The RED state: point the gate at the cached Store.GetSetting instead of
// GetMCPSettingsUncached and the two out-of-band flips stop biting.
func TestMCPMaskingToggleBitesTheNextRequest(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	f := setupMCPMaskingFixture(context.Background(), t)

	f.exemptFromMasking(t, "user:demo@example.com")
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"the session starts serving the caller's exemption")

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, f.session),
		"the toggle must bite the very next request of the session already open")

	f.writeMaskingToggleOutOfBand(t, false)
	a.Equal(maskedSecret, f.agentReadsSecret(t, f.session),
		"the gate must read the row rather than the profile it cached")

	f.writeMaskingToggleOutOfBand(t, true)
	a.Equal(maskPlaceholder, f.agentReadsSecret(t, f.session),
		"live in both directions, on the unchanged session")
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
// The control at the end is the JC1 pin: the guard is not the toggle's, so
// turning the toggle off still refuses. A clean statement through the same door
// is what proves the door is genuinely open — otherwise "refused" and "not
// reachable" look alike.
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

	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, true))

	// The console needs to see the real value, or "the write never ran" is
	// unprovable: a masked read shows the placeholder whether or not the UPDATE
	// landed. The exemption is suppressed for the agent by the toggle, which is
	// the other half of this fixture.
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
		"an ordinary change under the toggle is still served: %s", served.Error)

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

	// The control. Same session, same statement; only the toggle changed — and
	// the refusal does not, because masked reads happen without the toggle.
	a.NoError(f.ctl.setIgnoreMaskingExemptions(f.ctx, false))
	stillRefused := callAPIOnSession(f.ctx, t, f.session, "SheetService/CreateSheet",
		sheetBody(f.ctl.project.Name, maskedWrite))
	a.Equal(http.StatusForbidden, stillRefused.Status,
		"the guard is not the toggle's: %s", stillRefused.Error)
	stillRefusedQuery := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": maskedWrite,
	})
	a.Equal(http.StatusForbidden, stillRefusedQuery.Status,
		"on the execution door too: %s", stillRefusedQuery.Error)

	// And the door really is open, which "refused" alone cannot show. A clean
	// write through SQLService/Query is the only place in this PR an MCP session
	// runs DML end to end, so it is asserted on the stored value, not the status.
	const agentWrite = "UPDATE employee SET secret = 'rotated-by-agent' WHERE id = 1"
	agentRan := callAPIOnSession(f.ctx, t, f.session, "SQLService/Query", map[string]any{
		"name":      f.database,
		"statement": agentWrite,
	})
	a.Equal(http.StatusOK, agentRan.Status, "the execution door is open: %s", agentRan.Error)
	f.exemptFromMasking(t, "user:demo@example.com")
	a.Equal("rotated-by-agent", f.consoleReadsSecret(t),
		"the agent's write landed, so the refusals above were the guard and not a closed door")
}

// TestMCPSessionCannotFlipTheMaskingToggle closes the obvious way around the
// toggle: the agent turns it off first. UpdateSetting is FORBIDDEN to MCP
// sessions, so the refusal comes from the class gate ahead of the handler
// rather than from the caller's RBAC — and the principal here is a workspace
// admin, who would otherwise be allowed to write exactly this setting.
//
// The stored value is the load-bearing half: a guard that refused after the
// write would leave the workspace unprotected while reporting a denial.
func TestMCPSessionCannotFlipTheMaskingToggle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true))

	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()

	out := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", map[string]any{
		"allowMissing": true,
		"setting": map[string]any{
			"name": "settings/" + v1pb.Setting_MCP.String(),
			"value": map[string]any{
				"mcp": map[string]any{"ignoreMaskingExemptions": false},
			},
		},
		// A FieldMask crossing as JSON carries lowerCamel path segments; the
		// snake_case spelling the Go clients use is rejected before any
		// interceptor sees the request.
		"updateMask": "value.mcp.ignoreMaskingExemptions",
	})
	t.Logf("MCP UpdateSetting{ignore_masking_exemptions} → status=%d error=%q", out.Status, out.Error)

	a.Equal(http.StatusForbidden, out.Status,
		"an MCP session must not be able to rewrite the workspace settings")
	a.Contains(out.Error, "not available to MCP sessions",
		"the refusal must come from the FORBIDDEN gate, not from a permission check")

	stored, err := ctl.getIgnoreMaskingExemptions(ctx)
	a.NoError(err)
	a.True(stored, "the refusal must land before the write: the toggle is still on")

	// The operator's view. A gate denial runs before ACL, so it carries no
	// resource and its row is parented to the caller's workspace.
	rows := deniedMCPRows(ctx, t, ctl, workspace.Msg.Name, "/bytebase.v1.SettingService/UpdateSetting")
	a.NotEmpty(rows, "a denied settings write must be visible to an operator with MCP provenance")
	a.Equal(int32(connect.CodePermissionDenied), rows[0].Status.GetCode(),
		"the row must record the denial, not a success")
	a.NotEmpty(rows[0].McpDelegation.GetCorrelationId(),
		"the denial must be correlatable back to the agent session that made it")
}
