package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/sharepoint"
	"github.com/bytebase/bytebase/backend/store"
)

type fakeSharePointResolver struct {
	location *sharepoint.Location
	err      error
	resolved []string
}

func (c *fakeSharePointResolver) Resolve(_ context.Context, canonicalURL string) (*sharepoint.Location, error) {
	c.resolved = append(c.resolved, canonicalURL)
	return c.location, c.err
}

func (*fakeSharePointResolver) Deliver(context.Context, *sharepoint.Location, string, []sharepoint.File) (*sharepoint.Delivery, error) {
	return nil, errors.New("not used")
}

func TestResolveSharePointDestination(t *testing.T) {
	t.Parallel()
	const folderURL = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents/Exports"
	configured := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	newClient := func(client *fakeSharePointResolver) func(sharepoint.Config) (sharepoint.Client, error) {
		return func(cfg sharepoint.Config) (sharepoint.Client, error) {
			require.Equal(t, sharepoint.Config{TenantID: "tenant", ClientID: "client", ClientSecret: "secret"}, cfg)
			return client, nil
		}
	}

	t.Run("a URL that does not normalize is invalid", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{}
		_, err := resolveSharePointDestination(context.Background(), "not a url", configured, newClient(client))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Empty(t, client.resolved)
	})

	t.Run("an unconfigured workspace cannot resolve", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{}
		_, err := resolveSharePointDestination(context.Background(), folderURL, &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client"}, newClient(client))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.Empty(t, client.resolved)
	})

	t.Run("an unresolvable destination reports both possibilities", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{err: errors.Wrap(sharepoint.ErrNotFoundOrNoAccess, "403 accessDenied")}
		_, err := resolveSharePointDestination(context.Background(), folderURL, configured, newClient(client))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.Contains(t, err.Error(), "the SharePoint destination does not exist or Bytebase cannot access it")
	})

	t.Run("rejected credentials are reported", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{err: errors.Wrap(sharepoint.ErrCredential, "AADSTS7000215")}
		_, err := resolveSharePointDestination(context.Background(), folderURL, configured, newClient(client))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.Contains(t, err.Error(), sharepoint.ErrCredential.Error())
	})

	t.Run("a resolved destination stores the canonical URL and names", func(t *testing.T) {
		t.Parallel()
		canonicalURL, err := sharepoint.NormalizeURL(folderURL)
		require.NoError(t, err)
		client := &fakeSharePointResolver{location: &sharepoint.Location{
			DriveID:     "drive",
			ItemID:      "item",
			SiteName:    "Finance",
			LibraryName: "Documents",
			FolderPath:  "Exports",
			WebURL:      folderURL,
		}}
		destination, err := resolveSharePointDestination(context.Background(), folderURL, configured, newClient(client))
		require.NoError(t, err)
		require.Equal(t, []string{canonicalURL}, client.resolved)
		require.True(t, proto.Equal(&storepb.Project_ExportDestination{
			SharepointUrl: canonicalURL,
			SiteName:      "Finance",
			LibraryName:   "Documents",
			FolderPath:    "Exports",
			WebUrl:        folderURL,
		}, destination))
	})
}

func TestConvertToV1ExportDestination(t *testing.T) {
	t.Parallel()
	require.Nil(t, convertToV1ExportDestination(nil))
	require.Nil(t, convertToV1ExportDestination(&storepb.Project_ExportDestination{}), "a cleared destination is absent")
	require.True(t, proto.Equal(&v1pb.ExportDestination{
		SharepointUrl: "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents",
		SharepointLocation: &v1pb.SharePointLocation{
			SiteName:    "Finance",
			LibraryName: "Documents",
			WebUrl:      "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents",
		},
	}, convertToV1ExportDestination(&storepb.Project_ExportDestination{
		SharepointUrl: "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents",
		SiteName:      "Finance",
		LibraryName:   "Documents",
		WebUrl:        "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents",
	})))
}

func TestUpdateProjectExportDestination(t *testing.T) {
	t.Parallel()
	ctx, stores, projectID, _, _ := setupProjectInstanceLifecycleAPITest(t)
	stored := &storepb.Project_ExportDestination{
		SharepointUrl: "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents",
		SiteName:      "Finance",
		LibraryName:   "Documents",
	}
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{Workspace: "default", ResourceID: &projectID})
	require.NoError(t, err)
	setting := proto.CloneOf(project.Setting)
	setting.ExportDestination = stored
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{ResourceID: projectID, Workspace: "default", Setting: setting}))

	client := &fakeSharePointResolver{}
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := &ProjectService{store: stores, licenseService: licenseService, newSharePointClient: func(sharepoint.Config) (sharepoint.Client, error) { return client, nil }}
	update := func(rawURL string) (*v1pb.Project, error) {
		resp, err := service.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
			Project:    &v1pb.Project{Name: common.FormatProject(projectID), ExportDestination: &v1pb.ExportDestination{SharepointUrl: rawURL}},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"export_destination"}},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	storedDestination := func() *storepb.Project_ExportDestination {
		project, err := stores.GetProject(ctx, &store.FindProjectMessage{Workspace: "default", ResourceID: &projectID})
		require.NoError(t, err)
		return project.Setting.GetExportDestination()
	}

	_, err = update("https://contoso.sharepoint.com/sites/Finance/Shared%20Documents/Exports")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "setting a destination needs the license")
	require.Empty(t, client.resolved)
	require.True(t, proto.Equal(stored, storedDestination()))

	updated, err := update("")
	require.NoError(t, err, "clearing does not need the license")
	require.Nil(t, updated.ExportDestination)
	require.Nil(t, storedDestination())
}
