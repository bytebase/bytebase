package v1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/alexmullins/zip"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

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
	destination              v1pb.ExportDestination
	password                 string
	sharePointFeatureEnabled bool
	destinationURL           string
	sharePointSetting        *storepb.SharePointSetting
}

// decideExportDestination returns DOWNLOAD or SHAREPOINT. The workspace's
// disable_download is applied by the SQL Editor UI only.
func decideExportDestination(in exportDestinationInput) (v1pb.ExportDestination, error) {
	destination := in.destination
	if destination == v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED {
		destination = v1pb.ExportDestination_DOWNLOAD
	}
	switch destination {
	case v1pb.ExportDestination_DOWNLOAD:
		return v1pb.ExportDestination_DOWNLOAD, nil
	case v1pb.ExportDestination_SHAREPOINT:
	default:
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported export destination %v", destination))
	}
	if !in.sharePointFeatureEnabled {
		return 0, connect.NewError(connect.CodePermissionDenied, errors.Errorf("SharePoint export requires the %s feature", v1pb.PlanFeature_FEATURE_SHAREPOINT_EXPORT))
	}
	if in.password != "" {
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.New("password applies only to download"))
	}
	if in.destinationURL == "" {
		return 0, sharePointExportError(connect.CodeFailedPrecondition, errors.New("the project has no SharePoint export destination"), sharePointReasonNotConfigured, "")
	}
	setting := in.sharePointSetting
	if setting.GetTenantId() == "" || setting.GetClientId() == "" || setting.GetClientSecret() == "" {
		return 0, sharePointExportError(connect.CodeFailedPrecondition, errors.New("SharePoint is not configured in workspace settings"), sharePointReasonNotConfigured, "")
	}
	return v1pb.ExportDestination_SHAREPOINT, nil
}

// The ErrorInfo reasons of SharePoint export errors, which the SQL Editor
// explains to the user. Metadata "folder", when present, names the export
// folder where some files may already exist.
const (
	sharePointErrorDomain                = "bytebase.com"
	sharePointReasonNotConfigured        = "SHAREPOINT_NOT_CONFIGURED"
	sharePointReasonCredentialRejected   = "SHAREPOINT_CREDENTIAL_REJECTED"
	sharePointReasonDestinationNotFound  = "SHAREPOINT_DESTINATION_NOT_FOUND"
	sharePointReasonDestinationNotFolder = "SHAREPOINT_DESTINATION_NOT_FOLDER"
	sharePointReasonStorageFull          = "SHAREPOINT_STORAGE_FULL"
	sharePointReasonUploadUnconfirmed    = "SHAREPOINT_UPLOAD_UNCONFIRMED"
	sharePointReasonRequestFailed        = "SHAREPOINT_REQUEST_FAILED"
)

func sharePointExportError(code connect.Code, err error, reason, folder string) *connect.Error {
	connectErr := connect.NewError(code, err)
	info := &errdetails.ErrorInfo{Reason: reason, Domain: sharePointErrorDomain}
	if folder != "" {
		info.Metadata = map[string]string{"folder": folder}
	}
	if detail, detailErr := connect.NewErrorDetail(info); detailErr == nil {
		connectErr.AddDetail(detail)
	}
	return connectErr
}

func sharePointErrorReason(err error) string {
	switch {
	case errors.Is(err, sharepoint.ErrUnconfirmed):
		return sharePointReasonUploadUnconfirmed
	case errors.Is(err, sharepoint.ErrCredential):
		return sharePointReasonCredentialRejected
	case errors.Is(err, sharepoint.ErrNotFoundOrNoAccess):
		return sharePointReasonDestinationNotFound
	case errors.Is(err, sharepoint.ErrNotFolder):
		return sharePointReasonDestinationNotFolder
	case errors.Is(err, sharepoint.ErrStorageFull):
		return sharePointReasonStorageFull
	default:
		return sharePointReasonRequestFailed
	}
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
	in := exportDestinationInput{
		destination: request.GetDestination(),
		password:    request.GetPassword(),
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
		reason := sharePointReasonNotConfigured
		if errors.Is(err, sharepoint.ErrCredential) {
			reason = sharePointReasonCredentialRejected
		}
		return nil, sharePointExportError(connect.CodeFailedPrecondition, errors.Wrap(err, "invalid SharePoint configuration"), reason, "")
	}
	location, err := client.ResolveRecent(ctx, destinationURL)
	if err != nil {
		reason := sharePointErrorReason(err)
		if reason == sharePointReasonRequestFailed {
			return nil, sharePointExportError(connect.CodeInternal, errors.Wrap(err, "failed to resolve the SharePoint export destination"), reason, "")
		}
		return nil, sharePointExportError(connect.CodeFailedPrecondition, err, reason, "")
	}
	return &sharePointExportTarget{client: client, location: location}, nil
}

// uploadSharePointExport uploads files into folderName under the captured
// destination and returns the folder's web URL. A failure may leave some files
// uploaded, so every error names the folder.
func uploadSharePointExport(ctx context.Context, target *sharePointExportTarget, folderName string, files []sharepoint.File) (string, error) {
	folder, err := target.client.Upload(ctx, target.location, folderName, files)
	if err == nil {
		return folder.WebURL, nil
	}
	reason := sharePointErrorReason(err)
	if reason == sharePointReasonUploadUnconfirmed {
		return "", sharePointExportError(connect.CodeUnavailable, errors.Errorf("the upload to SharePoint could not be confirmed; files may exist in folder %q", folderName), reason, folderName)
	}
	filesMayRemainIn := folderName
	if errors.Is(err, sharepoint.ErrNothingUploaded) {
		filesMayRemainIn = ""
		err = errors.Wrap(err, "the upload to SharePoint failed; nothing was uploaded")
	} else {
		err = errors.Wrapf(err, "the upload to SharePoint failed; some files may already exist in folder %q", folderName)
	}
	if errors.Is(err, context.Canceled) {
		return "", connect.NewError(connect.CodeCanceled, err)
	}
	code := connect.CodeFailedPrecondition
	if reason == sharePointReasonRequestFailed {
		code = connect.CodeInternal
	}
	return "", sharePointExportError(code, err, reason, filesMayRemainIn)
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

// sendExport hands the export's files to their destination: a zip in the
// response for download, or an upload to the captured SharePoint target.
func sendExport(ctx context.Context, target *sharePointExportTarget, user *store.UserMessage, database *store.DatabaseMessage, files []sharepoint.File, request *v1pb.ExportRequest, exportTime time.Time) (*v1pb.ExportResponse, error) {
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
	uri, err := uploadSharePointExport(ctx, target, folderName, files)
	if err != nil {
		return nil, err
	}
	return &v1pb.ExportResponse{Uri: uri}, nil
}
