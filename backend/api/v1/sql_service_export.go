package v1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/export"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/sharepoint"
	"github.com/bytebase/bytebase/backend/store"
)

// exportDestinationInput holds the state an export's destination is decided on.
// The SharePoint fields are read only for the SharePoint destination.
type exportDestinationInput struct {
	destination v1pb.ExportDestination
	// allowedDestinations is the workspace query data policy's allow list.
	allowedDestinations      []storepb.ExportDestination
	password                 string
	sharePointFeatureEnabled bool
	destinationURL           string
	sharePointSetting        *storepb.SharePointSetting
}

// decideExportDestination returns DOWNLOAD or SHAREPOINT. Nothing unlocks a
// destination the policy does not allow: not an export grant, admin mode, or a
// lapsed license.
func decideExportDestination(in exportDestinationInput) (v1pb.ExportDestination, error) {
	destination := in.destination
	if destination == v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED {
		destination = v1pb.ExportDestination_DOWNLOAD
	}
	storeDestination, err := convertToStoreExportDestination(destination)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !slices.Contains(in.allowedDestinations, storeDestination) {
		return 0, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("export destination %s is not allowed by the workspace query data policy", destination))
	}
	if destination == v1pb.ExportDestination_DOWNLOAD {
		return v1pb.ExportDestination_DOWNLOAD, nil
	}
	if !in.sharePointFeatureEnabled {
		return 0, connect.NewError(connect.CodePermissionDenied, errors.Errorf("SharePoint export requires the %s feature", v1pb.PlanFeature_FEATURE_SHAREPOINT_EXPORT))
	}
	if in.password != "" {
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.New("password applies only to download"))
	}
	if in.destinationURL == "" {
		return 0, connect.NewError(connect.CodeFailedPrecondition, errors.New("the project has no SharePoint export destination"))
	}
	setting := in.sharePointSetting
	if setting.GetTenantId() == "" || setting.GetClientId() == "" || setting.GetClientSecret() == "" {
		return 0, connect.NewError(connect.CodeFailedPrecondition, errors.New("SharePoint is not configured in workspace settings"))
	}
	return v1pb.ExportDestination_SHAREPOINT, nil
}

// sharePointExportTarget is the destination an export captured when it started.
type sharePointExportTarget struct {
	client   sharepoint.Client
	location *sharepoint.Location
}

// prepareExportDestination decides the destination and, for SharePoint,
// resolves the project's destination so the export captures it before running
// the query.
func (s *SQLService) prepareExportDestination(ctx context.Context, request *v1pb.ExportRequest, database *store.DatabaseMessage) (*sharePointExportTarget, error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	// Read without the license check, so the policy outlives the license.
	policy, err := s.store.GetWorkspaceQueryDataPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get the workspace query data policy"))
	}
	in := exportDestinationInput{
		destination:         request.GetDestination(),
		allowedDestinations: store.GetAllowedExportDestinations(policy),
		password:            request.GetPassword(),
	}
	if in.destination == v1pb.ExportDestination_SHAREPOINT {
		in.sharePointFeatureEnabled = s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_SHAREPOINT_EXPORT) == nil
		project, err := s.store.GetProject(ctx, &store.FindProjectMessage{Workspace: workspaceID, ResourceID: &database.ProjectID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project %q", database.ProjectID))
		}
		if project == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", database.ProjectID))
		}
		in.destinationURL = project.Setting.GetExportSetting().GetSharepoint().GetUrl()
		exportSetting, err := s.store.GetExportSetting(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get export setting"))
		}
		in.sharePointSetting = exportSetting.GetSharepoint()
	}

	destination, err := decideExportDestination(in)
	if err != nil {
		return nil, err
	}
	if destination != v1pb.ExportDestination_SHAREPOINT {
		return nil, nil
	}
	return resolveSharePointExportTarget(ctx, s.sharePointClientFactory(), in.sharePointSetting, in.destinationURL)
}

func (s *SQLService) sharePointClientFactory() func(sharepoint.Config) (sharepoint.Client, error) {
	if s.newSharePointClient != nil {
		return s.newSharePointClient
	}
	return sharepoint.NewClient
}

func resolveSharePointExportTarget(ctx context.Context, newClient func(sharepoint.Config) (sharepoint.Client, error), setting *storepb.SharePointSetting, destinationURL string) (*sharePointExportTarget, error) {
	client, err := newClient(sharepoint.Config{
		TenantID:     setting.GetTenantId(),
		ClientID:     setting.GetClientId(),
		ClientSecret: setting.GetClientSecret(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Wrap(err, "invalid SharePoint configuration"))
	}
	location, err := client.Resolve(ctx, destinationURL)
	if err != nil {
		if errors.Is(err, sharepoint.ErrNotFoundOrNoAccess) || errors.Is(err, sharepoint.ErrCredential) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to resolve the SharePoint export destination"))
	}
	return &sharePointExportTarget{client: client, location: location}, nil
}

// deliverSharePointExport uploads files into folderName under the captured
// destination and returns the folder's web URL. A failure may leave some files
// delivered, so every error names the folder.
func deliverSharePointExport(ctx context.Context, target *sharePointExportTarget, folderName string, files []sharepoint.File) (string, error) {
	delivery, err := target.client.Deliver(ctx, target.location, folderName, files)
	if err != nil {
		if errors.Is(err, sharepoint.ErrUnconfirmed) {
			return "", connect.NewError(connect.CodeUnavailable, errors.Errorf("delivery to SharePoint could not be confirmed; files may exist in folder %q", folderName))
		}
		code := connect.CodeInternal
		if errors.Is(err, sharepoint.ErrNotFoundOrNoAccess) || errors.Is(err, sharepoint.ErrCredential) {
			code = connect.CodeFailedPrecondition
		}
		if errors.Is(err, sharepoint.ErrNothingDelivered) {
			return "", connect.NewError(code, errors.Wrap(err, "delivery to SharePoint failed; nothing was delivered"))
		}
		return "", connect.NewError(code, errors.Wrapf(err, "delivery to SharePoint failed; some files may already exist in folder %q", folderName))
	}
	return delivery.WebURL, nil
}

const (
	sharePointFolderComponentMaxLength = 64
	sharePointFolderNameMaxLength      = 200
)

// sharePointExportFolderName names the per-export folder. SharePoint attributes
// files to the application, so the name carries the exporter.
func sharePointExportFolderName(exportTime time.Time, exporterEmail, databaseName, exportID string) string {
	name := strings.Join([]string{
		exportTime.UTC().Format("20060102T150405Z"),
		sanitizeSharePointName(exporterEmail, sharePointFolderComponentMaxLength),
		sanitizeSharePointName(databaseName, sharePointFolderComponentMaxLength),
		exportID,
	}, "-")
	return sanitizeSharePointName(name, sharePointFolderNameMaxLength)
}

// sanitizeSharePointName replaces characters SharePoint rejects in file and
// folder names, trims the leading and trailing spaces and dots it rejects, and
// caps the length in runes.
func sanitizeSharePointName(name string, maxLength int) string {
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune("\"*:<>?/\\|#%", r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, " .")
	if runes := []rune(name); len(runes) > maxLength {
		name = strings.TrimRight(string(runes[:maxLength]), " .")
	}
	return name
}

func newExportID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "failed to generate export ID")
	}
	return hex.EncodeToString(b), nil
}

// buildExportFiles formats each result into its statement and result files.
func buildExportFiles(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, results []*v1pb.QueryResult, request *v1pb.ExportRequest) ([]sharepoint.File, error) {
	var files []sharepoint.File
	for i, result := range results {
		if result.GetError() != "" {
			return nil, errors.Errorf("failed to exec the SQL with error: %v", result.GetError())
		}
		baseName := fmt.Sprintf("statement-%d", i+1)
		var buf bytes.Buffer
		if err := formatExportResult(ctx, &buf, stores, instance, database, result, request); err != nil {
			return nil, errors.Wrap(err, "failed to write formatted result")
		}
		files = append(files,
			sharepoint.File{Name: baseName + ".sql", Content: []byte(result.Statement)},
			sharepoint.File{Name: fmt.Sprintf("%s.result.%s", baseName, strings.ToLower(request.Format.String())), Content: buf.Bytes()},
		)
		// Help GC by clearing the result data already formatted.
		result.Rows = nil
	}
	if len(files) == 0 {
		return nil, errors.Errorf("empty export data for database %s", database.DatabaseName)
	}
	return files, nil
}

// zipExportFiles packs files for download under <instance>/<database>/.
func zipExportFiles(database *store.DatabaseMessage, files []sharepoint.File, password string) ([]byte, error) {
	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)
	for _, file := range files {
		name := fmt.Sprintf("%s/%s/%s", database.InstanceID, database.DatabaseName, file.Name)
		if err := export.WriteZipEntry(zipw, name, file.Content, password); err != nil {
			return nil, err
		}
	}
	if err := zipw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close zip writer")
	}
	return buf.Bytes(), nil
}

// deliverExport hands the export's files to their destination: a zip in the
// response for download, or an upload to the captured SharePoint target.
func deliverExport(ctx context.Context, target *sharePointExportTarget, user *store.UserMessage, database *store.DatabaseMessage, files []sharepoint.File, request *v1pb.ExportRequest, exportTime time.Time) (*v1pb.ExportResponse, error) {
	if target == nil {
		content, err := zipExportFiles(database, files, request.GetPassword())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return &v1pb.ExportResponse{Content: content}, nil
	}

	exportID, err := newExportID()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	folderName := sharePointExportFolderName(exportTime, user.Email, database.DatabaseName, exportID)
	uri, err := deliverSharePointExport(ctx, target, folderName, files)
	if err != nil {
		return nil, err
	}
	return &v1pb.ExportResponse{Uri: uri}, nil
}
