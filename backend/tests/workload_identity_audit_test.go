package tests

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The exchange needs an HTTPS issuer on a public host, so the token is minted
// here the way ExchangeToken mints it. What this covers is everything after:
// the claim reaching the audit row, the API and the search filter.
func TestWorkloadIdentityCallsCarryFederatedIdentity(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startProject(ctx, t)
	workspace := getCurrentWorkspace(ctx, t, ctl)
	workspaceID, err := common.GetWorkspaceID(workspace)
	a.NoError(err)

	identityID := generateRandomString("wi")
	identity, err := ctl.workloadIdentityServiceClient.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
		Parent:             workspace,
		WorkloadIdentityId: identityID,
		WorkloadIdentity: &v1pb.WorkloadIdentity{
			Title: "Federated audit identity",
			WorkloadIdentityConfig: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				IssuerUrl:        "https://token.actions.githubusercontent.com",
				AllowedAudiences: []string{"bytebase"},
				SubjectPattern:   "repo:my-org/*",
			},
		},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, workspace, common.WorkloadIdentityBindingPrefix+identity.Msg.Email, "roles/workspaceAdmin")
	a.NoError(err)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()
	var authSecret string
	a.NoError(metadataDB.QueryRowContext(ctx, `SELECT payload->>'authSecret' FROM server_config LIMIT 1`).Scan(&authSecret))

	// The subject is unique to this test: the server and workspace are shared.
	federated := common.FederatedIdentity{
		Issuer:  "https://token.actions.githubusercontent.com",
		Subject: "repo:my-org/" + identityID + ":ref:refs/heads/main",
	}
	identityToken, err := auth.GenerateWorkloadIdentityToken(identity.Msg.Email, workspaceID, federated, authSecret)
	a.NoError(err)

	rename := func(title string) {
		_, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
			Project:    &v1pb.Project{Name: ctl.project.Name, Title: title},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
		a.NoError(err)
	}
	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = identityToken
	rename("renamed by the workload identity")
	ctl.authInterceptor.token = adminToken
	rename("renamed by the admin")

	search := func(filter string) []*v1pb.AuditLog {
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent: ctl.project.Name,
			Filter: filter,
		}))
		a.NoError(err)
		return resp.Msg.AuditLogs
	}

	federatedRows := search(`federated_subject == "` + federated.Subject + `"`)
	a.Len(federatedRows, 1)
	a.Equal("/bytebase.v1.ProjectService/UpdateProject", federatedRows[0].Method)
	a.Equal(common.FormatWorkloadIdentityEmail(identity.Msg.Email), federatedRows[0].Actor)
	a.Equal(federated.Issuer, federatedRows[0].GetFederatedIdentity().GetIssuer())
	a.Equal(federated.Subject, federatedRows[0].GetFederatedIdentity().GetSubject())

	var withoutFederatedIdentity int
	for _, row := range search(`method == "/bytebase.v1.ProjectService/UpdateProject"`) {
		if row.GetFederatedIdentity() == nil {
			withoutFederatedIdentity++
		}
	}
	a.Equal(1, withoutFederatedIdentity, "the admin's call carries no federated identity")
}
