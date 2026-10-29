package v1

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/export"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/sharepoint"
	"github.com/bytebase/bytebase/backend/store"
)

func TestDecideExportDestination(t *testing.T) {
	t.Parallel()
	configured := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	download := []storepb.ExportDestination{storepb.ExportDestination_DOWNLOAD}
	sharePointOnly := []storepb.ExportDestination{storepb.ExportDestination_SHAREPOINT}
	sharePointReady := func(in exportDestinationInput) exportDestinationInput {
		in.destination = v1pb.ExportDestination_SHAREPOINT
		if in.allowedDestinations == nil {
			in.allowedDestinations = sharePointOnly
		}
		in.sharePointFeatureEnabled = true
		in.destinationURL = "https://contoso.sharepoint.com/sites/data/Exports"
		in.sharePointSetting = configured
		return in
	}
	for _, tc := range []struct {
		name     string
		in       exportDestinationInput
		want     v1pb.ExportDestination
		wantCode connect.Code
	}{
		{name: "unspecified is download", in: exportDestinationInput{allowedDestinations: download}, want: v1pb.ExportDestination_DOWNLOAD},
		{name: "download", in: exportDestinationInput{destination: v1pb.ExportDestination_DOWNLOAD, allowedDestinations: download, password: "pw"}, want: v1pb.ExportDestination_DOWNLOAD},
		{name: "download not allowed", in: exportDestinationInput{destination: v1pb.ExportDestination_DOWNLOAD, allowedDestinations: sharePointOnly}, wantCode: connect.CodeFailedPrecondition},
		{name: "unspecified not allowed", in: exportDestinationInput{allowedDestinations: sharePointOnly}, wantCode: connect.CodeFailedPrecondition},
		{name: "nothing allowed", in: exportDestinationInput{destination: v1pb.ExportDestination_DOWNLOAD}, wantCode: connect.CodeFailedPrecondition},
		{name: "sharepoint", in: sharePointReady(exportDestinationInput{}), want: v1pb.ExportDestination_SHAREPOINT},
		{name: "sharepoint not allowed", in: sharePointReady(exportDestinationInput{allowedDestinations: download}), wantCode: connect.CodeFailedPrecondition},
		{
			name: "sharepoint without the feature",
			in: func() exportDestinationInput {
				in := sharePointReady(exportDestinationInput{})
				in.sharePointFeatureEnabled = false
				return in
			}(),
			wantCode: connect.CodePermissionDenied,
		},
		{
			name: "sharepoint with password",
			in: func() exportDestinationInput {
				in := sharePointReady(exportDestinationInput{})
				in.password = "pw"
				return in
			}(),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "sharepoint without destination",
			in: func() exportDestinationInput {
				in := sharePointReady(exportDestinationInput{})
				in.destinationURL = ""
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name: "sharepoint without credentials",
			in: func() exportDestinationInput {
				in := sharePointReady(exportDestinationInput{})
				in.sharePointSetting = nil
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
		{
			name: "sharepoint without client secret",
			in: func() exportDestinationInput {
				in := sharePointReady(exportDestinationInput{})
				in.sharePointSetting = &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client"}
				return in
			}(),
			wantCode: connect.CodeFailedPrecondition,
		},
		{name: "unknown destination", in: exportDestinationInput{destination: v1pb.ExportDestination(99), allowedDestinations: []storepb.ExportDestination{storepb.ExportDestination_DOWNLOAD, storepb.ExportDestination_SHAREPOINT}}, wantCode: connect.CodeInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := decideExportDestination(tc.in)
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

// TestZipExportFilesMatchesStreamedZip compares the download zip with the one the
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
	user := &store.UserMessage{Email: "alice@example.com"}
	database := &store.DatabaseMessage{InstanceID: "prod", DatabaseName: "hr"}
	exportTime := time.Date(2026, 10, 22, 1, 5, 7, 0, time.UTC)
	resp, err := deliverExport(ctx, target, user, database, files, &v1pb.ExportRequest{Format: v1pb.ExportFormat_CSV}, exportTime)
	require.NoError(t, err)
	require.Empty(t, resp.Content)
	require.Equal(t, "https://contoso.sharepoint.com/sites/data/Exports/"+client.deliveredFolder, resp.Uri)
	require.Same(t, client.location, client.deliveredTo)
	require.Regexp(t, regexp.MustCompile("^20261022T010507Z-alice@example.com-hr-[0-9a-f]{8}$"), client.deliveredFolder)
	require.Equal(t, files, client.deliveredFiles)
}

// nothingDeliveredError stands in for the SharePoint client's marking of a
// delivery that failed before any upload.
type nothingDeliveredError struct{ error }

func (e nothingDeliveredError) Unwrap() error      { return e.error }
func (nothingDeliveredError) Is(target error) bool { return target == sharepoint.ErrNothingDelivered }

func TestSharePointExportErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	setting := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	for _, tc := range []struct {
		name         string
		resolveErr   error
		deliverErr   error
		wantCode     connect.Code
		wantInMsg    []string
		wantNotInMsg []string
	}{
		{name: "resolve not found", resolveErr: errors.Wrap(sharepoint.ErrNotFoundOrNoAccess, "graph"), wantCode: connect.CodeFailedPrecondition},
		{name: "resolve credential", resolveErr: sharepoint.ErrCredential, wantCode: connect.CodeFailedPrecondition},
		{name: "resolve other", resolveErr: errors.New("boom"), wantCode: connect.CodeInternal},
		{name: "deliver not found", deliverErr: sharepoint.ErrNotFoundOrNoAccess, wantCode: connect.CodeFailedPrecondition, wantInMsg: []string{"may already exist", "20261022T010507Z-alice@example.com-hr-0a1b2c3d"}},
		{name: "folder not created", deliverErr: nothingDeliveredError{sharepoint.ErrNotFoundOrNoAccess}, wantCode: connect.CodeFailedPrecondition, wantInMsg: []string{"nothing was delivered"}, wantNotInMsg: []string{"may already exist", "20261022T010507Z-alice@example.com-hr-0a1b2c3d"}},
		{name: "deliver credential", deliverErr: errors.Wrap(sharepoint.ErrCredential, "token"), wantCode: connect.CodeFailedPrecondition, wantInMsg: []string{"may already exist", "20261022T010507Z-alice@example.com-hr-0a1b2c3d"}},
		{name: "deliver unconfirmed", deliverErr: errors.Wrap(sharepoint.ErrUnconfirmed, "upload"), wantCode: connect.CodeUnavailable, wantInMsg: []string{"could not be confirmed", "20261022T010507Z-alice@example.com-hr-0a1b2c3d"}},
		{name: "deliver failed after some files", deliverErr: errors.New("upload statement-1.result.csv: 507 insufficientStorage"), wantCode: connect.CodeInternal, wantInMsg: []string{"delivery to SharePoint failed", "may already exist", "insufficientStorage", "20261022T010507Z-alice@example.com-hr-0a1b2c3d"}},
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
			for _, want := range tc.wantInMsg {
				require.Contains(t, err.Error(), want)
			}
			for _, unwanted := range tc.wantNotInMsg {
				require.NotContains(t, err.Error(), unwanted)
			}
		})
	}
}

// newEnterpriseLicenseService returns a license service under which
// workspaceID holds a valid ENTERPRISE license.
func newEnterpriseLicenseService(ctx context.Context, t *testing.T, stores *store.Store, workspaceID string) *enterprise.LicenseService {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, string(privateKeyPEM))
	require.NoError(t, err)
	setLicenseServicePublicKey(t, licenseService, &privateKey.PublicKey)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_SYSTEM, Workspace: workspaceID, Value: &storepb.SystemSetting{}})
	require.NoError(t, err)
	license, err := licenseService.CreateLicense(&enterprise.LicenseParams{Plan: v1pb.PlanType_ENTERPRISE.String(), Seats: 10, Instances: 10, WorkspaceID: workspaceID})
	require.NoError(t, err)
	require.NoError(t, licenseService.StoreLicense(ctx, workspaceID, license))
	require.NoError(t, licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_SHAREPOINT_EXPORT))
	return licenseService
}

func setWorkspaceAllowedExportDestinations(ctx context.Context, t *testing.T, stores *store.Store, destinations ...storepb.ExportDestination) {
	t.Helper()
	payload, err := protojson.Marshal(&storepb.QueryDataPolicy{
		ExportDestinationPolicy: &storepb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: destinations},
	})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_WORKSPACE,
		Resource:     common.FormatWorkspace("default"),
		Type:         storepb.Policy_QUERY_DATA,
		Payload:      string(payload),
		Enforce:      true,
	})
	require.NoError(t, err)
}

// newExportDestinationFixture seeds a project whose SharePoint destination and
// workspace credentials are configured.
func newExportDestinationFixture(t *testing.T) (context.Context, *store.Store, *store.DatabaseMessage) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO project (resource_id, workspace, name, setting)
    VALUES ('export-project', 'default', 'Export Project', '{"exportSetting":{"sharepoint":{"url":"https://contoso.sharepoint.com/sites/data/Exports"}}}');
    INSERT INTO setting (name, workspace, value)
    VALUES ('EXPORT', 'default', '{"sharepoint":{"tenantId":"tenant","clientId":"client","clientSecret":"secret"}}');`)
	require.NoError(t, err)
	return ctx, stores, &store.DatabaseMessage{ProjectID: "export-project", InstanceID: "prod", DatabaseName: "hr"}
}

// TestPrepareExportDestinationReadsPolicyWithoutLicense runs on a free plan, as
// after a lapsed license: the export destination policy still applies.
func TestPrepareExportDestinationReadsPolicyWithoutLicense(t *testing.T) {
	t.Parallel()
	ctx, stores, database := newExportDestinationFixture(t)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	require.Error(t, licenseService.IsFeatureEnabled(ctx, "default", v1pb.PlanFeature_FEATURE_QUERY_POLICY))
	service := &SQLService{store: stores, licenseService: licenseService}

	target, err := service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: v1pb.ExportDestination_DOWNLOAD}, database)
	require.NoError(t, err, "a workspace without a policy allows download")
	require.Nil(t, target)

	setWorkspaceAllowedExportDestinations(ctx, t, stores, storepb.ExportDestination_SHAREPOINT)
	for _, destination := range []v1pb.ExportDestination{v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED, v1pb.ExportDestination_DOWNLOAD} {
		_, err := service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: destination}, database)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), destination.String())
	}
	_, err = service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: v1pb.ExportDestination_SHAREPOINT}, database)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "the SharePoint destination needs the license")

	setWorkspaceAllowedExportDestinations(ctx, t, stores, storepb.ExportDestination_DOWNLOAD)
	target, err = service.prepareExportDestination(ctx, &v1pb.ExportRequest{}, database)
	require.NoError(t, err)
	require.Nil(t, target)
}

func TestPrepareExportDestinationAllowsOnlyListedDestinations(t *testing.T) {
	t.Parallel()
	ctx, stores, database := newExportDestinationFixture(t)
	licenseService := newEnterpriseLicenseService(ctx, t, stores, "default")
	client := &exportSharePointClient{location: &sharepoint.Location{DriveID: "drive", ItemID: "item"}}
	service := &SQLService{store: stores, licenseService: licenseService, newSharePointClient: func(cfg sharepoint.Config) (sharepoint.Client, error) {
		require.Equal(t, sharepoint.Config{TenantID: "tenant", ClientID: "client", ClientSecret: "secret"}, cfg)
		return client, nil
	}}

	_, err := service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: v1pb.ExportDestination_SHAREPOINT}, database)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "SharePoint is not allowed by default")
	require.Empty(t, client.resolvedURL)

	setWorkspaceAllowedExportDestinations(ctx, t, stores, storepb.ExportDestination_SHAREPOINT)
	target, err := service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: v1pb.ExportDestination_SHAREPOINT}, database)
	require.NoError(t, err)
	require.Same(t, client.location, target.location)
	require.Equal(t, "https://contoso.sharepoint.com/sites/data/Exports", client.resolvedURL)

	_, err = service.prepareExportDestination(ctx, &v1pb.ExportRequest{Destination: v1pb.ExportDestination_DOWNLOAD}, database)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "the license does not unlock download")
}

// TestExportRejectsDownloadDespiteExportGrant calls Export with an active
// export grant for the exact statement. The policy rejects the request before
// any connection to the target database.
func TestExportRejectsDownloadDespiteExportGrant(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{Email: "alice@example.com"})
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
    INSERT INTO workspace (resource_id) VALUES ('default');
    INSERT INTO project (resource_id, workspace, name, setting)
    VALUES ('grant-project', 'default', 'Grant Project', '{"allowJustInTimeAccess":true}');`)
	require.NoError(t, err)
	instance, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			Activation:  true,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	database, err := stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "prod", DatabaseName: "hr", ProjectID: "grant-project", Metadata: &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	const statement = "SELECT 1"
	expires := time.Now().Add(time.Hour)
	grant, err := stores.CreateAccessGrant(ctx, &store.AccessGrantMessage{
		ProjectID: "grant-project", Creator: "alice@example.com", Status: storepb.AccessGrant_ACTIVE, ExpireTime: &expires,
		Payload: &storepb.AccessGrantPayload{Query: statement, Targets: []string{"instances/prod/databases/hr"}, Export: true, Unmask: true},
	})
	require.NoError(t, err)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := &SQLService{store: stores, licenseService: licenseService}
	got := service.preCheckAccess(ctx, statement, instance, database, nil, "", true /* requireExport */)
	require.NotNil(t, got)
	require.Equal(t, grant.ID, got.ID)

	setWorkspaceAllowedExportDestinations(ctx, t, stores, storepb.ExportDestination_SHAREPOINT)
	for _, destination := range []v1pb.ExportDestination{v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED, v1pb.ExportDestination_DOWNLOAD} {
		_, err := service.Export(ctx, connect.NewRequest(&v1pb.ExportRequest{
			Name:        "instances/prod/databases/hr",
			Statement:   statement,
			Format:      v1pb.ExportFormat_CSV,
			Destination: destination,
		}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), destination.String())
		require.ErrorContains(t, err, "export destination DOWNLOAD is not allowed", destination.String())
	}
}
