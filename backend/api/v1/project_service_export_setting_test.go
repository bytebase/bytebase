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

func (*fakeSharePointResolver) Upload(context.Context, *sharepoint.Location, string, []sharepoint.File) (*sharepoint.Folder, error) {
	return nil, errors.New("not used")
}

func TestResolveSharePointURL(t *testing.T) {
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
		_, err := resolveSharePointURL(context.Background(), "not a url", configured, newClient(client))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Empty(t, client.resolved)
	})

	t.Run("missing credentials cannot resolve", func(t *testing.T) {
		t.Parallel()
		for _, setting := range []*storepb.SharePointSetting{nil, {TenantId: "tenant", ClientId: "client"}} {
			client := &fakeSharePointResolver{}
			_, err := resolveSharePointURL(context.Background(), folderURL, setting, newClient(client))
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.Empty(t, client.resolved)
		}
	})

	t.Run("an unresolvable destination reports both possibilities", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{err: errors.Wrap(sharepoint.ErrNotFoundOrNoAccess, "403 accessDenied")}
		_, err := resolveSharePointURL(context.Background(), folderURL, configured, newClient(client))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.Contains(t, err.Error(), "the SharePoint destination does not exist or Bytebase cannot access it")
	})

	t.Run("rejected credentials are reported", func(t *testing.T) {
		t.Parallel()
		client := &fakeSharePointResolver{err: errors.Wrap(sharepoint.ErrCredential, "AADSTS7000215")}
		_, err := resolveSharePointURL(context.Background(), folderURL, configured, newClient(client))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.Contains(t, err.Error(), sharepoint.ErrCredential.Error())
	})

	t.Run("a resolved destination returns the canonical URL", func(t *testing.T) {
		t.Parallel()
		canonicalURL, err := sharepoint.NormalizeURL(folderURL)
		require.NoError(t, err)
		client := &fakeSharePointResolver{location: &sharepoint.Location{DriveID: "drive", ItemID: "item", WebURL: folderURL}}
		got, err := resolveSharePointURL(context.Background(), folderURL, configured, newClient(client))
		require.NoError(t, err)
		require.Equal(t, canonicalURL, got)
		require.Equal(t, []string{canonicalURL}, client.resolved)
	})
}

func TestConvertToV1ProjectExportSetting(t *testing.T) {
	t.Parallel()
	require.Nil(t, convertToV1ProjectExportSetting(nil))
	require.Nil(t, convertToV1ProjectExportSetting(&storepb.Project_ExportSetting{}), "a cleared setting is absent")
	require.Nil(t, convertToV1ProjectExportSetting(&storepb.Project_ExportSetting{Sharepoint: &storepb.Project_SharePointDestination{}}))
	require.True(t, proto.Equal(&v1pb.Project_ExportSetting{
		Sharepoint: &v1pb.SharePointDestination{Url: "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents"},
	}, convertToV1ProjectExportSetting(&storepb.Project_ExportSetting{
		Sharepoint: &storepb.Project_SharePointDestination{Url: "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents"},
	})))
}

func TestUpdateProjectExportSetting(t *testing.T) {
	t.Parallel()
	const rawURL = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents/Forms/AllItems.aspx?id=%2Fsites%2FFinance%2FShared%20Documents%2FExports"
	canonicalURL, err := sharepoint.NormalizeURL(rawURL)
	require.NoError(t, err)
	require.NotEqual(t, rawURL, canonicalURL)

	ctx, stores, projectID, _, _ := setupProjectInstanceLifecycleAPITest(t)
	client := &fakeSharePointResolver{location: &sharepoint.Location{DriveID: "drive", ItemID: "item"}}
	newService := func(licenseService *enterprise.LicenseService) *ProjectService {
		return &ProjectService{store: stores, licenseService: licenseService, newSharePointClient: func(sharepoint.Config) (sharepoint.Client, error) { return client, nil }}
	}
	update := func(service *ProjectService, exportSetting *v1pb.Project_ExportSetting) (*v1pb.Project, error) {
		resp, err := service.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
			Project:    &v1pb.Project{Name: common.FormatProject(projectID), ExportSetting: exportSetting},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"export_setting"}},
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	withURL := func(url string) *v1pb.Project_ExportSetting {
		return &v1pb.Project_ExportSetting{Sharepoint: &v1pb.SharePointDestination{Url: url}}
	}
	storedSetting := func() *storepb.Project_ExportSetting {
		project, err := stores.GetProject(ctx, &store.FindProjectMessage{Workspace: "default", ResourceID: &projectID})
		require.NoError(t, err)
		return project.Setting.GetExportSetting()
	}

	licensed := newService(newEnterpriseLicenseService(ctx, t, stores, "default"))
	_, err = update(licensed, withURL(rawURL))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "the workspace has no SharePoint credentials")
	require.Empty(t, client.resolved)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_EXPORT,
		Workspace: "default",
		Value:     &storepb.ExportSetting{Sharepoint: &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}},
	})
	require.NoError(t, err)

	_, err = update(licensed, withURL("not a url"))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	updated, err := update(licensed, withURL(rawURL))
	require.NoError(t, err)
	require.True(t, proto.Equal(withURL(canonicalURL), updated.ExportSetting))
	require.Equal(t, []string{canonicalURL}, client.resolved)
	require.True(t, proto.Equal(&storepb.Project_ExportSetting{
		Sharepoint: &storepb.Project_SharePointDestination{Url: canonicalURL},
	}, storedSetting()), "only the canonical URL is stored")

	client.err = sharepoint.ErrNotFoundOrNoAccess
	_, err = update(licensed, withURL("https://contoso.sharepoint.com/sites/Finance/Shared%20Documents/Gone"))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	client.err = nil

	freeLicenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	unlicensed := newService(freeLicenseService)
	_, err = update(unlicensed, withURL(rawURL))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "setting a destination needs the license")
	require.True(t, proto.Equal(&storepb.Project_ExportSetting{
		Sharepoint: &storepb.Project_SharePointDestination{Url: canonicalURL},
	}, storedSetting()))

	for _, cleared := range []*v1pb.Project_ExportSetting{withURL(""), nil} {
		updated, err = update(unlicensed, cleared)
		require.NoError(t, err, "clearing does not need the license")
		require.Nil(t, updated.ExportSetting)
		require.Nil(t, storedSetting())
	}
}
