package v1

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/export"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/sharepoint"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheckExportApproval(t *testing.T) {
	t.Parallel()
	grant := &store.AccessGrantMessage{Payload: &storepb.AccessGrantPayload{Export: true}}
	for _, tc := range []struct {
		name          string
		disableExport bool
		grant         *store.AccessGrantMessage
		wantDenied    bool
	}{
		{name: "export allowed by policy"},
		{name: "export allowed by policy with grant", grant: grant},
		{name: "approval required without grant", disableExport: true, wantDenied: true},
		{name: "approval required with export grant", disableExport: true, grant: grant},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkExportApproval(&store.EffectiveQueryDataPolicy{DisableExport: tc.disableExport}, tc.grant)
			if tc.wantDenied {
				require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDecideExportDelivery(t *testing.T) {
	t.Parallel()
	configured := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	sharePointReady := func(in exportDeliveryInput) exportDeliveryInput {
		in.delivery = v1pb.ExportRequest_SHAREPOINT
		in.sharePointFeatureEnabled = true
		in.destinationURL = "https://contoso.sharepoint.com/sites/data/Exports"
		in.sharePointSetting = configured
		return in
	}
	for _, tc := range []struct {
		name     string
		in       exportDeliveryInput
		want     v1pb.ExportRequest_Delivery
		wantCode connect.Code
	}{
		{name: "unspecified is local", in: exportDeliveryInput{}, want: v1pb.ExportRequest_LOCAL},
		{name: "local", in: exportDeliveryInput{delivery: v1pb.ExportRequest_LOCAL, password: "pw"}, want: v1pb.ExportRequest_LOCAL},
		{name: "local download disabled", in: exportDeliveryInput{delivery: v1pb.ExportRequest_LOCAL, disableLocalExportDownload: true}, wantCode: connect.CodePermissionDenied},
		{name: "unspecified with local download disabled", in: exportDeliveryInput{disableLocalExportDownload: true}, wantCode: connect.CodePermissionDenied},
		{
			name: "license lapsed with local download disabled leaves no delivery",
			in: func() exportDeliveryInput {
				in := sharePointReady(exportDeliveryInput{disableLocalExportDownload: true})
				in.sharePointFeatureEnabled = false
				return in
			}(),
			wantCode: connect.CodePermissionDenied,
		},
		{name: "sharepoint", in: sharePointReady(exportDeliveryInput{}), want: v1pb.ExportRequest_SHAREPOINT},
		{name: "sharepoint with local download disabled", in: sharePointReady(exportDeliveryInput{disableLocalExportDownload: true}), want: v1pb.ExportRequest_SHAREPOINT},
		{
			name: "sharepoint with password",
			in: func() exportDeliveryInput {
				in := sharePointReady(exportDeliveryInput{})
				in.password = "pw"
				return in
			}(),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "sharepoint without destination",
			in: func() exportDeliveryInput {
				in := sharePointReady(exportDeliveryInput{})
				in.destinationURL = ""
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name: "sharepoint without setting",
			in: func() exportDeliveryInput {
				in := sharePointReady(exportDeliveryInput{})
				in.sharePointSetting = nil
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name: "sharepoint without client secret",
			in: func() exportDeliveryInput {
				in := sharePointReady(exportDeliveryInput{})
				in.sharePointSetting = &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client"}
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := decideExportDelivery(tc.in)
			if tc.wantCode != 0 {
				require.Equal(t, tc.wantCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSanitizeSharePointName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		input     string
		maxLength int
		want      string
	}{
		{name: "plain", input: "alice@example.com", maxLength: 64, want: "alice@example.com"},
		{name: "reserved characters", input: "a\"b*c:d<e>f?g/h\\i|j#k%l", maxLength: 64, want: "a_b_c_d_e_f_g_h_i_j_k_l"},
		{name: "control characters", input: "a\tb\nc", maxLength: 64, want: "a_b_c"},
		{name: "leading and trailing spaces and dots", input: " ..db.. ", maxLength: 64, want: "db"},
		{name: "capped in runes", input: "数据库数据库", maxLength: 3, want: "数据库"},
		{name: "cap exposes trailing dot", input: "abc.def", maxLength: 4, want: "abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, sanitizeSharePointName(tc.input, tc.maxLength))
		})
	}
}

func TestSharePointExportFolderName(t *testing.T) {
	t.Parallel()
	exportTime := time.Date(2026, 10, 22, 9, 5, 7, 0, time.FixedZone("CST", 8*60*60))
	require.Equal(t,
		"20261022T010507Z-alice@example.com-hr_prod-0a1b2c3d",
		sharePointExportFolderName(exportTime, "alice@example.com", "hr/prod", "0a1b2c3d"),
	)

	long := sharePointExportFolderName(exportTime, strings.Repeat("e", 300)+"@example.com", strings.Repeat("d", 300), "0a1b2c3d")
	require.LessOrEqual(t, len([]rune(long)), sharePointFolderNameMaxLength)
	require.True(t, strings.HasSuffix(long, "-0a1b2c3d"), long)

	id, err := newExportID()
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile("^[0-9a-f]{8}$"), id)
}

// TestZipExportFilesMatchesStreamedZip compares the local zip with the one the
// export wrote before files were built first: entries were streamed directly
// into the zip under <instance>/<database>/.
func TestZipExportFilesMatchesStreamedZip(t *testing.T) {
	t.Parallel()
	database := &store.DatabaseMessage{InstanceID: "prod-instance", DatabaseName: "hr"}
	files := []sharepoint.File{
		{Name: "statement-1.sql", Content: []byte("SELECT * FROM employee")},
		{Name: "statement-1.result.csv", Content: []byte(strings.Repeat("1,alice\n2,bob\n", 5000))},
		{Name: "statement-2.sql", Content: []byte("SELECT 1")},
		{Name: "statement-2.result.csv", Content: []byte("1\n")},
	}
	for _, password := range []string{"", "secret"} {
		t.Run("password="+password, func(t *testing.T) {
			t.Parallel()
			got, err := zipExportFiles(database, files, password)
			require.NoError(t, err)
			want := streamedZip(t, database, files, password)

			gotEntries, wantEntries := readZip(t, got, password), readZip(t, want, password)
			require.Equal(t, []string{
				"prod-instance/hr/statement-1.sql",
				"prod-instance/hr/statement-1.result.csv",
				"prod-instance/hr/statement-2.sql",
				"prod-instance/hr/statement-2.result.csv",
			}, zipEntryNames(gotEntries))
			require.Equal(t, wantEntries, gotEntries)
			if password == "" {
				require.Equal(t, len(want), len(got))
			}
		})
	}
}

type zipEntry struct {
	name             string
	method           uint16
	encrypted        bool
	crc32            uint32
	compressedSize   uint64
	uncompressedSize uint64
	content          []byte
}

func zipEntryNames(entries []zipEntry) []string {
	var names []string
	for _, e := range entries {
		names = append(names, e.name)
	}
	return names
}

func streamedZip(t *testing.T, database *store.DatabaseMessage, files []sharepoint.File, password string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)
	for i, file := range files {
		name := database.InstanceID + "/" + database.DatabaseName + "/" + file.Name
		if i%2 == 0 {
			require.NoError(t, export.WriteZipEntry(zipw, name, file.Content, password))
			continue
		}
		w, err := export.CreateZipWriter(zipw, name, password)
		require.NoError(t, err)
		for content := file.Content; len(content) > 0; {
			n := min(len(content), 4096)
			_, err := w.Write(content[:n])
			require.NoError(t, err)
			content = content[n:]
		}
	}
	require.NoError(t, zipw.Close())
	return buf.Bytes()
}

func readZip(t *testing.T, content []byte, password string) []zipEntry {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	var entries []zipEntry
	for _, f := range r.File {
		if f.IsEncrypted() {
			f.SetPassword(password)
		}
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		entries = append(entries, zipEntry{
			name:             f.Name,
			method:           f.Method,
			encrypted:        f.IsEncrypted(),
			crc32:            f.CRC32,
			compressedSize:   f.CompressedSize64,
			uncompressedSize: f.UncompressedSize64,
			content:          data,
		})
	}
	return entries
}

type exportSharePointClient struct {
	location   *sharepoint.Location
	resolveErr error
	deliverErr error

	resolvedURL     string
	deliveredTo     *sharepoint.Location
	deliveredFolder string
	deliveredFiles  []sharepoint.File
}

func (c *exportSharePointClient) Resolve(_ context.Context, canonicalURL string) (*sharepoint.Location, error) {
	c.resolvedURL = canonicalURL
	if c.resolveErr != nil {
		return nil, c.resolveErr
	}
	return c.location, nil
}

func (c *exportSharePointClient) Deliver(_ context.Context, loc *sharepoint.Location, folderName string, files []sharepoint.File) (*sharepoint.Delivery, error) {
	c.deliveredTo, c.deliveredFolder, c.deliveredFiles = loc, folderName, files
	if c.deliverErr != nil {
		return nil, c.deliverErr
	}
	return &sharepoint.Delivery{FolderName: folderName, WebURL: loc.WebURL + "/" + folderName}, nil
}

func TestSharePointExportDeliversToCapturedTarget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &exportSharePointClient{location: &sharepoint.Location{DriveID: "drive", ItemID: "item", WebURL: "https://contoso.sharepoint.com/sites/data/Exports"}}
	var gotConfig sharepoint.Config
	newClient := func(cfg sharepoint.Config) (sharepoint.Client, error) {
		gotConfig = cfg
		return client, nil
	}
	setting := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	target, err := resolveSharePointExportTarget(ctx, newClient, setting, "https://contoso.sharepoint.com/sites/data/Exports")
	require.NoError(t, err)
	require.Equal(t, sharepoint.Config{TenantID: "tenant", ClientID: "client", ClientSecret: "secret"}, gotConfig)
	require.Equal(t, "https://contoso.sharepoint.com/sites/data/Exports", client.resolvedURL)

	files := []sharepoint.File{
		{Name: "statement-1.sql", Content: []byte("SELECT 1")},
		{Name: "statement-1.result.csv", Content: []byte("1\n")},
	}
	service := &SQLService{}
	user := &store.UserMessage{Email: "alice@example.com"}
	database := &store.DatabaseMessage{InstanceID: "prod", DatabaseName: "hr"}
	exportTime := time.Date(2026, 10, 22, 1, 5, 7, 0, time.UTC)
	resp, err := service.deliverExport(ctx, target, user, database, files, &v1pb.ExportRequest{Format: v1pb.ExportFormat_CSV}, exportTime)
	require.NoError(t, err)
	require.Empty(t, resp.Content)
	require.Equal(t, client.deliveredFolder, resp.SharepointDelivery.GetFolderName())
	require.Equal(t, "https://contoso.sharepoint.com/sites/data/Exports/"+client.deliveredFolder, resp.SharepointDelivery.GetWebUrl())
	require.Same(t, client.location, client.deliveredTo)
	require.Regexp(t, regexp.MustCompile("^20261022T010507Z-alice@example.com-hr-[0-9a-f]{8}$"), client.deliveredFolder)
	require.Equal(t, files, client.deliveredFiles)
}

func TestSharePointExportErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	setting := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	for _, tc := range []struct {
		name       string
		resolveErr error
		deliverErr error
		wantCode   connect.Code
		wantInMsg  []string
	}{
		{name: "resolve not found", resolveErr: errors.Wrap(sharepoint.ErrNotFoundOrNoAccess, "graph"), wantCode: connect.CodeFailedPrecondition},
		{name: "resolve credential", resolveErr: sharepoint.ErrCredential, wantCode: connect.CodeFailedPrecondition},
		{name: "resolve other", resolveErr: errors.New("boom"), wantCode: connect.CodeInternal},
		{name: "deliver not found", deliverErr: sharepoint.ErrNotFoundOrNoAccess, wantCode: connect.CodeFailedPrecondition, wantInMsg: []string{"may already exist"}},
		{name: "deliver credential", deliverErr: errors.Wrap(sharepoint.ErrCredential, "token"), wantCode: connect.CodeFailedPrecondition, wantInMsg: []string{"may already exist"}},
		{name: "deliver unconfirmed", deliverErr: errors.Wrap(sharepoint.ErrUnconfirmed, "upload"), wantCode: connect.CodeUnavailable, wantInMsg: []string{"could not be confirmed"}},
		{name: "deliver failed after some files", deliverErr: errors.New("upload statement-1.result.csv: 507 insufficientStorage"), wantCode: connect.CodeInternal, wantInMsg: []string{"delivery to SharePoint failed", "may already exist", "insufficientStorage"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &exportSharePointClient{location: &sharepoint.Location{WebURL: "https://contoso.sharepoint.com/sites/data/Exports"}, resolveErr: tc.resolveErr, deliverErr: tc.deliverErr}
			newClient := func(sharepoint.Config) (sharepoint.Client, error) { return client, nil }
			target, err := resolveSharePointExportTarget(ctx, newClient, setting, "https://contoso.sharepoint.com/sites/data/Exports")
			if tc.resolveErr != nil {
				require.Equal(t, tc.wantCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			_, err = deliverSharePointExport(ctx, target, "20261022T010507Z-alice@example.com-hr-0a1b2c3d", nil)
			require.Equal(t, tc.wantCode, connect.CodeOf(err))
			require.Contains(t, err.Error(), "20261022T010507Z-alice@example.com-hr-0a1b2c3d")
			for _, want := range tc.wantInMsg {
				require.Contains(t, err.Error(), want)
			}
		})
	}
}

// TestLocalExportDownloadSwitchIgnoresLicense runs on a free plan, as after a
// lapsed license: the switch still blocks local download, and SharePoint
// delivery is unavailable.
func TestLocalExportDownloadSwitchIgnoresLicense(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO project (resource_id, workspace, name, setting)
    VALUES ('export-project', 'default', 'Export Project', '{"exportDestination":{"sharepointUrl":"https://contoso.sharepoint.com/sites/data/Exports"}}');
    INSERT INTO setting (name, workspace, value)
    VALUES ('SHAREPOINT', 'default', '{"tenantId":"tenant","clientId":"client","clientSecret":"secret"}');`)
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	require.Error(t, licenseService.IsFeatureEnabled(ctx, "default", v1pb.PlanFeature_FEATURE_SHAREPOINT_EXPORT))
	service := &SQLService{store: stores, licenseService: licenseService}
	database := &store.DatabaseMessage{ProjectID: "export-project", InstanceID: "prod", DatabaseName: "hr"}
	setSwitch := func(disabled bool) {
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
			Name: storepb.SettingName_WORKSPACE_PROFILE, Workspace: "default",
			Value: &storepb.WorkspaceProfileSetting{DisableLocalExportDownload: disabled},
		})
		require.NoError(t, err)
	}

	setSwitch(false)
	target, err := service.prepareExportDelivery(ctx, &v1pb.ExportRequest{}, database)
	require.NoError(t, err)
	require.Nil(t, target)

	// The switch turned on while the export ran blocks the bytes.
	setSwitch(true)
	files := []sharepoint.File{{Name: "statement-1.sql", Content: []byte("SELECT 1")}}
	_, err = service.deliverExport(ctx, nil, &store.UserMessage{Email: "alice@example.com"}, database, files, &v1pb.ExportRequest{}, time.Now())
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	for _, delivery := range []v1pb.ExportRequest_Delivery{v1pb.ExportRequest_DELIVERY_UNSPECIFIED, v1pb.ExportRequest_LOCAL, v1pb.ExportRequest_SHAREPOINT} {
		_, err := service.prepareExportDelivery(ctx, &v1pb.ExportRequest{Delivery: delivery}, database)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), delivery.String())
	}
}
