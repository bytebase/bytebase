package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSearchAPIListServices(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test listing all services (no parameters)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Available Services")
	require.Contains(t, text, "SQLService")
	require.Contains(t, text, "DatabaseService")
	available, unavailable, ok := strings.Cut(text, "## Not available over MCP")
	require.True(t, ok, "the list names what MCP cannot do")
	// A service the MCP ceiling serves nothing of is not on the menu, only on
	// the list of what MCP cannot do. IdentityProviderService is the clearest
	// case: its writes choose what will later be trusted to mint a credential
	// and its reads are workspace administration, so every one of its methods
	// is refused.
	require.NotContains(t, available, "IdentityProviderService")
	require.Contains(t, unavailable, "**IdentityProviderService**")
	// ProjectService is the mixed case, on the menu with only its reads: an
	// agent that cannot name a project cannot reach a database at all.
	require.Contains(t, available, "ProjectService")
}

func TestSearchAPIByService(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test browsing a specific service
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "SQLService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/")
	require.Contains(t, text, "endpoints")
	// Browse mode should not include schema
	require.NotContains(t, text, "Request Body")
}

func TestSearchAPIByServiceNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test browsing a non-existent service
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "NonExistentService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown service: NonExistentService")
	require.Contains(t, text, "Available Services")
}

func TestSearchAPIDetailMode(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test detail mode with full operationId
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "bytebase.v1.SQLService.Query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/Query")
	require.Contains(t, text, "Request Body")
	require.Contains(t, text, "```json")
}

func TestSearchAPIDetailModeShortFormat(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test detail mode with short operationId format (Service/Method)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "SQLService/Query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SQLService/Query")
	require.Contains(t, text, "Request Body")
	require.Contains(t, text, "```json")
}

func TestSearchAPIDetailModeWithResponse(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test detail mode shows response schema
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "bytebase.v1.DatabaseService.ListDatabases",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Response Body")
}

func TestSearchAPIDetailModeNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test detail mode with unknown operationId
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "unknown.operation.id",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown operationId: unknown.operation.id")
}

func TestSearchAPIServiceShowsAll(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test browsing a service shows all endpoints (no limit)
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Service: "DatabaseService",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	// Should show "Found X endpoints" not "Showing X of Y"
	require.Contains(t, text, "Found")
	require.NotContains(t, text, "Showing")
}

func TestSearchAPISchemaLookup(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test schema lookup with full name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "bytebase.v1.Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "\"name\":")
	require.Contains(t, text, "\"engine\":")
}

func TestSearchAPISchemaLookupShortName(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test schema lookup with short name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "bytebase.v1.Instance")
	require.Contains(t, text, "\"name\":")
}

func TestSearchAPISchemaLookupNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test schema lookup with unknown name
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "NonExistentSchema",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Unknown schema")
}

func TestSearchAPISchemaLookupEnum(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test enum schema lookup
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		Schema: "Engine",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Enum values:")
}

// Schema mode cuts each description to its first 97 characters, so the part
// of RolloutPolicy.roles that agents misread has to come first.
func TestSearchAPIRolloutPolicyNamesWhoApproves(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{Schema: "RolloutPolicy"})
	require.NoError(t, err)
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, `"roles": array<string> // Who can run, skip or cancel the environment's rollout tasks; WORKSPACE_APPROVAL sets approvers.`)
}

func TestSearchAPIProtobufDescriptionTruncation(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test that protobuf types have short descriptions
	result, _, err := s.handleSearchAPI(context.Background(), nil, SearchInput{
		OperationID: "InstanceService/CreateInstance",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(*mcpsdk.TextContent).Text

	// Should NOT contain verbose protobuf documentation
	require.NotContains(t, text, "A Timestamp represents a point in time")
	require.NotContains(t, text, "A Duration represents a signed")

	// Should contain short description if there's a timestamp field
	if strings.Contains(text, "google.protobuf.Timestamp") {
		require.Contains(t, text, "ISO 8601")
	}
}

func newSearchServer(t *testing.T) *Server {
	t.Helper()
	s, err := newServerWithStore(newTestServerStore(), &config.Profile{Mode: common.ReleaseModeDev}, "test-secret", nil)
	require.NoError(t, err)
	return s
}

func searchText(t *testing.T, s *Server, input SearchInput) string {
	t.Helper()
	result, _, err := s.handleSearchAPI(context.Background(), nil, input)
	require.NoError(t, err)
	require.Len(t, result.Content, 1)
	return result.Content[0].(*mcpsdk.TextContent).Text
}

func TestSearchAPINamesWhatMCPCannotDo(t *testing.T) {
	s := newSearchServer(t)
	administers := auth.MCPDenialSentence(v1pb.MCPMethodClass_EXCLUDED, v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE)
	mintsForOthers := auth.MCPDenialSentence(v1pb.MCPMethodClass_FORBIDDEN, v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS)

	t.Run("the service list names each unavailable service, why, and the console", func(t *testing.T) {
		_, unavailable, ok := strings.Cut(searchText(t, s, SearchInput{}), "## Not available over MCP")
		require.True(t, ok)
		require.Contains(t, unavailable, "whatever the user's permissions")
		require.Contains(t, unavailable, "Bytebase console")
		require.NotContains(t, unavailable, "Request Body")
		_, block, ok := strings.Cut(unavailable, "- **ServiceAccountService**")
		require.True(t, ok)
		block, _, _ = strings.Cut(block, "\n- **")
		require.Equal(t, ": manages service accounts for API integrations.\n"+
			"  - Refused because "+mintsForOthers+".\n"+
			"  - Refused because "+administers+".", strings.TrimSuffix(block, "\n"), "the service's reasons sit under it")
	})

	t.Run("a service MCP cannot call is not reported as missing", func(t *testing.T) {
		text := searchText(t, s, SearchInput{Service: "ServiceAccountService"})
		require.Contains(t, text, "ServiceAccountService is not available over MCP")
		require.Contains(t, text, "- ServiceAccountService/CreateServiceAccount, ServiceAccountService/UpdateServiceAccount: refused because "+mintsForOthers+".\n",
			"operations refused for one reason share one line")
		require.NotContains(t, text, "Unknown service")
		require.NotContains(t, text, "Request Body")
	})

	t.Run("a served service also names the operations MCP cannot call", func(t *testing.T) {
		text := searchText(t, s, SearchInput{Service: "InstanceService"})
		served, unavailable, ok := strings.Cut(text, "## Not available over MCP")
		require.True(t, ok)
		require.Contains(t, served, "InstanceService/GetInstance")
		require.NotContains(t, served, "InstanceService/DeleteInstance")
		require.Contains(t, unavailable, "InstanceService/DeleteInstance")
		require.Equal(t, 1, strings.Count(unavailable, "refused because "+administers), "one line per reason")
	})

	t.Run("the workload identity create names the tool and its fields", func(t *testing.T) {
		text := searchText(t, s, SearchInput{OperationID: "WorkloadIdentityService/CreateWorkloadIdentity"})
		for _, want := range []string{
			"through the administer_workspace tool", "workloadIdentityId", "providerType GITHUB or GITLAB",
			`\"repo:{owner}/{repo}:*\"`, "without it or the repository's ID", "no approval rule holds them", "Request Body",
		} {
			require.Contains(t, text, strings.ReplaceAll(want, `\"`, `"`))
		}
	})

	t.Run("the project policy write names the one change MCP makes", func(t *testing.T) {
		text := searchText(t, s, SearchInput{OperationID: "ProjectService/SetIamPolicy"})
		require.Contains(t, text, "only when administer_workspace grants a workload identity it creates")
		require.Contains(t, text, "change other members, roles and conditions signed in to the Bytebase console")
	})

	t.Run("detail mode gives the refusal, not the schema", func(t *testing.T) {
		text := searchText(t, s, SearchInput{OperationID: "InstanceService/DeleteInstance"})
		require.Contains(t, text, "## InstanceService/DeleteInstance")
		require.Contains(t, text, "Not available over MCP")
		require.Contains(t, text, administers)
		require.Contains(t, text, "Bytebase console")
		require.NotContains(t, text, "Request Body")
		require.NotContains(t, text, "Response Body")
		require.NotContains(t, text, "Unknown operationId")
	})

	t.Run("the setting read names the settings it serves", func(t *testing.T) {
		const readable = "settings/WORKSPACE_APPROVAL, settings/ENVIRONMENT, settings/SEMANTIC_TYPES, settings/DATA_CLASSIFICATION"
		text := searchText(t, s, SearchInput{Service: "SettingService"})
		served, unavailable, ok := strings.Cut(text, "## Not available over MCP")
		require.True(t, ok)
		require.Contains(t, served, "SettingService/GetSetting")
		require.Contains(t, served, readable)
		require.Contains(t, served, "own Bytebase permissions still apply")
		require.Contains(t, served, `returns "setting <NAME> not found"`)
		require.Contains(t, served, "bb.default (full mask) and bb.default-partial (partial mask) always exist")
		for _, method := range []string{"ListSettings", "TestEmailSetting"} {
			require.Contains(t, unavailable, "SettingService/"+method)
		}
		require.Contains(t, served, "SettingService/UpdateSetting")
		require.Contains(t, served, "change only the approval rules for database changes, through the administer_workspace tool")
		require.NotContains(t, unavailable, "SettingService/UpdateSetting")

		detail := searchText(t, s, SearchInput{OperationID: "SettingService/GetSetting"})
		require.Contains(t, detail, readable)
		require.Contains(t, detail, "Request Body")

		detail = searchText(t, s, SearchInput{OperationID: "SettingService/UpdateSetting"})
		require.Contains(t, detail, "through the administer_workspace tool")
		require.Contains(t, detail, "Request Body")
	})

	t.Run("the policy update names the tools that change policies", func(t *testing.T) {
		detail := searchText(t, s, SearchInput{OperationID: "OrgPolicyService/UpdatePolicy"})
		require.Contains(t, detail, "the workspace masking rules through the administer_workspace tool")
		require.Contains(t, detail, "grant_masking_exemption and revoke_masking_exemption")
		require.Contains(t, detail, "Request Body")
	})

	t.Run("the instance create names the tool that adds instances", func(t *testing.T) {
		text := searchText(t, s, SearchInput{Service: "InstanceService"})
		served, unavailable, ok := strings.Cut(text, "## Not available over MCP")
		require.True(t, ok)
		require.Contains(t, served, "InstanceService/CreateInstance")
		require.NotContains(t, unavailable, "InstanceService/CreateInstance")

		detail := searchText(t, s, SearchInput{OperationID: "InstanceService/CreateInstance"})
		require.Contains(t, detail, "add a PostgreSQL instance that signs in with a user name and password to a project, through the administer_workspace tool")
		require.Contains(t, detail, "Request Body")
	})
}

// TestSearchAPINamesEveryRefusedMethod derives the refused methods from the
// annotations, so a method refused later is named with the gate's reason
// without anyone listing it here.
func TestSearchAPINamesEveryRefusedMethod(t *testing.T) {
	type refusedMethod struct {
		service, method, reason string
	}
	var refused []refusedMethod
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		for i := range fd.Services().Len() {
			sd := fd.Services().Get(i)
			for j := range sd.Methods().Len() {
				if md := sd.Methods().Get(j); md.IsStreamingClient() || md.IsStreamingServer() {
					// The spec has no path for a streaming RPC, so neither search nor
					// call_api can reach one.
					continue
				}
				procedure := fmt.Sprintf("/%s/%s", sd.FullName(), sd.Methods().Get(j).Name())
				class, reason, err := auth.MCPAnnotationsOfProcedure(procedure)
				require.NoError(t, err)
				if !auth.MCPClassIsRefused(class) {
					continue
				}
				refused = append(refused, refusedMethod{string(sd.Name()), string(sd.Methods().Get(j).Name()), auth.MCPDenialSentence(class, reason)})
			}
		}
		return true
	})
	require.NotEmpty(t, refused)

	s := newSearchServer(t)
	listings := map[string]string{}
	for _, m := range refused {
		if _, ok := listings[m.service]; !ok {
			listings[m.service] = searchText(t, s, SearchInput{Service: m.service})
		}
		require.Contains(t, listings[m.service], fmt.Sprintf("%s/%s", m.service, m.method))
		require.Contains(t, listings[m.service], "refused because "+m.reason)

		detail := searchText(t, s, SearchInput{OperationID: m.service + "/" + m.method})
		require.Contains(t, detail, m.reason, "%s/%s", m.service, m.method)
		require.NotContains(t, detail, "Request Body", "%s/%s", m.service, m.method)
	}
}

func TestSearchAPIShowsBytesAsBase64(t *testing.T) {
	s := newSearchServer(t)
	text := searchText(t, s, SearchInput{Schema: "Release.File"})
	require.Contains(t, text, `"statement": string (base64-encoded bytes)`)
	require.Contains(t, text, `"path": string //`, "other strings keep their plain type")
}
