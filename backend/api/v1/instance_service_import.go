package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/neon"
)

const (
	importEntryLimit  = 20
	importWorkerCount = 4
	importErrorDomain = "bytebase.com"
)

// ErrorInfo reasons shared by vendor discovery and ImportInstances. No reason
// names a vendor; a vendor condition carries it in the "vendor" metadata key.
const (
	reasonInvalidPageToken          = "INVALID_PAGE_TOKEN"
	reasonDestinationNotFound       = "DESTINATION_NOT_FOUND"
	reasonRequestCanceled           = "REQUEST_CANCELED"
	reasonRequestDeadlineExceeded   = "REQUEST_DEADLINE_EXCEEDED"
	reasonVendorCredentialInvalid   = "VENDOR_CREDENTIAL_INVALID"
	reasonVendorAccessDenied        = "VENDOR_ACCESS_DENIED"
	reasonVendorResourceNotFound    = "VENDOR_RESOURCE_NOT_FOUND"
	reasonVendorRateLimited         = "VENDOR_RATE_LIMITED"
	reasonVendorUnavailable         = "VENDOR_UNAVAILABLE"
	reasonVendorResponseInvalid     = "VENDOR_RESPONSE_INVALID"
	reasonVendorEndpointUnavailable = "VENDOR_ENDPOINT_UNAVAILABLE"
	reasonVendorUserNotFound        = "VENDOR_USER_NOT_FOUND"
	reasonVendorDatabaseNotFound    = "VENDOR_DATABASE_NOT_FOUND"
	reasonVendorPasswordRequired    = "VENDOR_PASSWORD_REQUIRED"
	reasonInstanceConnectionFailed  = "INSTANCE_CONNECTION_FAILED"
	reasonInstanceSettingsInvalid   = "INSTANCE_SETTINGS_INVALID"
	reasonInstanceValidationFailed  = "INSTANCE_VALIDATION_FAILED"
	reasonInstanceIDExists          = "INSTANCE_ID_EXISTS"
	reasonInstanceLimitExceeded     = "INSTANCE_LIMIT_EXCEEDED"
	reasonInstanceCreationFailed    = "INSTANCE_CREATION_FAILED"
)

// importTarget is what a vendor source resolves to.
type importTarget struct {
	engine     v1pb.Engine
	vendor     v1pb.Vendor
	dataSource *v1pb.DataSource
}

// importOperations is the seam between the vendor-neutral runner and the
// vendor resolver. resolve returns errors already mapped to the import error
// model.
type importOperations struct {
	resolve func(context.Context, *v1pb.InstanceImport) (*importTarget, error)
	create  func(context.Context, *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error)
}

// ImportInstances returns an error only before any creation starts. The
// frontend treats INVALID_ARGUMENT, UNAUTHENTICATED, PERMISSION_DENIED, and
// NOT_FOUND on this method as proof that nothing was created, so no code path
// after admission may return one of them as an RPC error.
func (s *InstanceService) ImportInstances(ctx context.Context, req *connect.Request[v1pb.ImportInstancesRequest]) (*connect.Response[v1pb.ImportInstancesResponse], error) {
	if err := validateImportRequest(req.Msg); err != nil {
		return nil, err
	}
	if _, err := s.getProjectInstanceParent(ctx, req.Msg.Parent); err != nil {
		return nil, importDestinationError(err)
	}
	var resolve func(context.Context, *v1pb.InstanceImport) (*importTarget, error)
	switch credential := req.Msg.Credential.(type) {
	case *v1pb.ImportInstancesRequest_NeonCredential:
		resolve = neonImportResolver(neon.NewClient(credential.NeonCredential.GetApiKey()))
	default:
		return nil, newBadRequestError(fieldViolation("credential", "a credential is required"))
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	activation := importActivation(s.licenseService.IsUnifiedInstanceLicense(ctx, workspaceID), s.licenseService.GetActivatedInstanceLimit(ctx, workspaceID))
	results := runImport(ctx, req.Msg, activation, importOperations{
		resolve: resolve,
		create: func(ctx context.Context, request *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
			return s.createInstanceWithSync(ctx, request, false)
		},
	})
	return connect.NewResponse(&v1pb.ImportInstancesResponse{Results: results}), nil
}

// importActivation reports whether imported instances request activation.
// A license with no activated-instance allowance rejects every creation that
// requests it, so those workspaces import without activation.
func importActivation(unifiedLicense bool, activatedInstanceLimit int) bool {
	return unifiedLicense || activatedInstanceLimit > 0
}

func validateImportRequest(request *v1pb.ImportInstancesRequest) error {
	var violations []*errdetails.BadRequest_FieldViolation
	add := func(field, description string) {
		violations = append(violations, fieldViolation(field, description))
	}
	neonCredential := false
	switch credential := request.Credential.(type) {
	case *v1pb.ImportInstancesRequest_NeonCredential:
		neonCredential = true
		if err := validateNeonAPIKey(credential.NeonCredential.GetApiKey()); err != nil {
			add("neon_credential.api_key", err.Error())
		}
	default:
		add("credential", "a credential is required")
	}
	if len(request.Imports) < 1 || len(request.Imports) > importEntryLimit {
		add("imports", fmt.Sprintf("must contain between 1 and %d entries", importEntryLimit))
		return newBadRequestError(violations...)
	}
	instanceIDs := map[string]int{}
	sources := map[string]int{}
	for i, entry := range request.Imports {
		field := func(name string) string { return fmt.Sprintf("imports[%d].%s", i, name) }
		if entry == nil {
			add(fmt.Sprintf("imports[%d]", i), "an entry is required")
			continue
		}
		if !common.IsValidResourceID(entry.InstanceId) {
			add(field("instance_id"), "must be a valid resource ID")
		} else if first, ok := instanceIDs[entry.InstanceId]; ok {
			add(field("instance_id"), fmt.Sprintf("duplicates imports[%d].instance_id", first))
		} else {
			instanceIDs[entry.InstanceId] = i
		}
		if instance := entry.Instance; instance == nil {
			add(field("instance"), "an instance is required")
		} else {
			if strings.TrimSpace(instance.Title) == "" {
				add(field("instance.title"), "a title is required")
			}
			if interval := instance.SyncInterval; interval != nil && (interval.CheckValid() != nil || interval.AsDuration() < 0) {
				add(field("instance.sync_interval"), "must be a valid nonnegative duration")
			}
			// Accepting one of these would let an entry contradict its source.
			if instance.Engine != v1pb.Engine_ENGINE_UNSPECIFIED {
				add(field("instance.engine"), "must be unset; the source determines the engine")
			}
			if instance.Vendor != v1pb.Vendor_VENDOR_UNSPECIFIED {
				add(field("instance.vendor"), "must be unset; the source determines the vendor")
			}
			if len(instance.DataSources) > 0 {
				add(field("instance.data_sources"), "must be empty; the source determines the data source")
			}
		}
		if entry.Username == "" {
			add(field("username"), "a username is required")
		}
		if entry.ConnectionDatabase == "" {
			add(field("connection_database"), "a connection database is required")
		}
		if entry.Password != nil && *entry.Password == "" {
			add(field("password"), "a supplied password must not be empty")
		}
		var sourceKey string
		switch source := entry.Source.(type) {
		case *v1pb.InstanceImport_NeonBranch:
			branch := source.NeonBranch
			switch {
			case !neonCredential:
				add(field("neon_branch"), "a Neon branch requires neon_credential")
			case !neon.ValidResourceID(branch.GetProjectId()):
				add(field("neon_branch.project_id"), "must be a valid Neon project ID")
			case !neon.ValidResourceID(branch.GetBranchId()):
				add(field("neon_branch.branch_id"), "must be a valid Neon branch ID")
			default:
				sourceKey = "neon_branch/" + branch.GetProjectId() + "/" + branch.GetBranchId()
			}
		default:
			add(field("source"), "a source is required")
		}
		if sourceKey == "" {
			continue
		}
		if first, ok := sources[sourceKey]; ok {
			add(field("source"), fmt.Sprintf("duplicates the source of imports[%d]", first))
		} else {
			sources[sourceKey] = i
		}
	}
	if len(violations) > 0 {
		return newBadRequestError(violations...)
	}
	return nil
}

func runImport(ctx context.Context, request *v1pb.ImportInstancesRequest, activation bool, operations importOperations) []*v1pb.InstanceImportResult {
	results := make([]*v1pb.InstanceImportResult, len(request.Imports))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for range importWorkerCount {
		workers.Go(func() {
			for i := range jobs {
				results[i] = importInstance(ctx, request.Parent, request.Imports[i], activation, operations)
			}
		})
	}
	for i := range request.Imports {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	return results
}

func importInstance(ctx context.Context, parent *string, entry *v1pb.InstanceImport, activation bool, operations importOperations) *v1pb.InstanceImportResult {
	if err := ctx.Err(); err != nil {
		return notCreated(requestInterruptedError(err))
	}
	target, err := operations.resolve(ctx, entry)
	if err != nil {
		return notCreated(err)
	}
	candidate := proto.CloneOf(entry.Instance)
	candidate.Engine = target.engine
	candidate.Vendor = target.vendor
	candidate.DataSources = []*v1pb.DataSource{target.dataSource}
	candidate.Activation = activation
	create := connect.NewRequest(&v1pb.CreateInstanceRequest{Parent: parent, InstanceId: entry.InstanceId, Instance: candidate, ValidateOnly: true})
	if _, err := operations.create(ctx, create); err != nil {
		return notCreated(instanceCreateError(err))
	}
	if err := ctx.Err(); err != nil {
		return notCreated(requestInterruptedError(err))
	}
	create.Msg.ValidateOnly = false
	instance, err := operations.create(ctx, create)
	if err != nil {
		// These codes are returned only by checks that run before the instance
		// row is written. Any other failure may follow a committed write.
		switch connect.CodeOf(err) {
		case connect.CodeAlreadyExists, connect.CodeInvalidArgument, connect.CodeResourceExhausted, connect.CodePermissionDenied, connect.CodeNotFound:
			return notCreated(instanceCreateError(err))
		default:
			return creationUnknown(err)
		}
	}
	return &v1pb.InstanceImportResult{Outcome: &v1pb.InstanceImportResult_Imported{Imported: instance.Msg}}
}

func notCreated(err error) *v1pb.InstanceImportResult {
	return &v1pb.InstanceImportResult{Outcome: &v1pb.InstanceImportResult_Failure{Failure: &v1pb.InstanceImportFailure{Status: importStatus(err), CreationOutcome: v1pb.InstanceImportFailure_NOT_CREATED}}}
}

// creationUnknown is the only source of INSTANCE_CREATION_FAILED, so that
// reason and the UNKNOWN outcome always appear together. It keeps the code of
// err and drops its message, which can carry the host and user name.
func creationUnknown(err error) *v1pb.InstanceImportResult {
	status := importStatus(newReasonError(connect.CodeOf(err), reasonInstanceCreationFailed, "Bytebase could not confirm whether this instance was created. Check the instance list before retrying.", nil))
	return &v1pb.InstanceImportResult{Outcome: &v1pb.InstanceImportResult_Failure{Failure: &v1pb.InstanceImportFailure{Status: status, CreationOutcome: v1pb.InstanceImportFailure_UNKNOWN}}}
}

// instanceCreateError maps a CreateInstance failure that precedes the write
// to the import error model. A connection failure keeps only its category:
// driver messages can carry the host and user name.
func instanceCreateError(err error) *connect.Error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		if category := connectErr.Meta().Get(connectionCategoryHeader); category != "" {
			return newReasonError(connect.CodeFailedPrecondition, reasonInstanceConnectionFailed, "Bytebase could not connect with these settings", map[string]string{"category": category})
		}
	}
	var missingEnvironment *environmentNotFoundError
	if errors.As(err, &missingEnvironment) {
		return newReasonError(connect.CodeInvalidArgument, reasonInstanceSettingsInvalid, missingEnvironment.Error(), nil)
	}
	switch code := connect.CodeOf(err); code {
	case connect.CodeResourceExhausted:
		return newReasonError(code, reasonInstanceLimitExceeded, "The workspace instance or activated-instance limit has been reached. Remove instances that are no longer needed or raise the limit, then retry.", nil)
	case connect.CodeAlreadyExists:
		return newReasonError(code, reasonInstanceIDExists, "This instance ID already exists. Check the existing instance or choose a different ID.", nil)
	case connect.CodeNotFound:
		return newReasonError(code, reasonDestinationNotFound, "The destination no longer exists.", nil)
	case connect.CodeInvalidArgument:
		return newReasonError(code, reasonInstanceSettingsInvalid, connectErr.Message(), nil)
	case connect.CodePermissionDenied:
		return newReasonError(connect.CodeInvalidArgument, reasonInstanceSettingsInvalid, "The instance settings require permissions or subscription features that are unavailable.", nil)
	case connect.CodeCanceled:
		return requestInterruptedError(context.Canceled)
	case connect.CodeDeadlineExceeded:
		return requestInterruptedError(context.DeadlineExceeded)
	default:
		return newReasonError(code, reasonInstanceValidationFailed, "Bytebase could not check these instance settings, so nothing was created. Retry.", nil)
	}
}

func requestInterruptedError(err error) *connect.Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newReasonError(connect.CodeDeadlineExceeded, reasonRequestDeadlineExceeded, "The request deadline passed before this entry was processed", nil)
	}
	return newReasonError(connect.CodeCanceled, reasonRequestCanceled, "The request was canceled before this entry was processed", nil)
}

// importDestinationError marks a missing destination and names the field of a
// malformed one. Other errors pass through.
func importDestinationError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeNotFound:
		return newReasonError(connect.CodeNotFound, reasonDestinationNotFound, errorMessage(err), nil)
	case connect.CodeInvalidArgument:
		return newBadRequestError(fieldViolation("parent", errorMessage(err)))
	default:
		return err
	}
}

func errorMessage(err error) string {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Message()
	}
	return err.Error()
}

func newReasonError(code connect.Code, reason, message string, metadata map[string]string) *connect.Error {
	err := connect.NewError(code, errors.New(message))
	if detail, e := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: reason, Domain: importErrorDomain, Metadata: metadata}); e == nil {
		err.AddDetail(detail)
	}
	return err
}

func newVendorError(code connect.Code, reason, message string, vendor v1pb.Vendor) *connect.Error {
	return newReasonError(code, reason, message, map[string]string{"vendor": vendor.String()})
}

func fieldViolation(field, description string) *errdetails.BadRequest_FieldViolation {
	return &errdetails.BadRequest_FieldViolation{Field: field, Description: description}
}

// newBadRequestError reports malformed input as INVALID_ARGUMENT with one
// BadRequest field violation per problem and no ErrorInfo reason.
func newBadRequestError(violations ...*errdetails.BadRequest_FieldViolation) *connect.Error {
	message := "invalid request"
	if len(violations) > 0 {
		message = fmt.Sprintf("%s: %s", violations[0].Field, violations[0].Description)
	}
	err := connect.NewError(connect.CodeInvalidArgument, errors.New(message))
	if detail, e := connect.NewErrorDetail(&errdetails.BadRequest{FieldViolations: violations}); e == nil {
		err.AddDetail(detail)
	}
	return err
}

func importStatus(err error) *statuspb.Status {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		connectErr = connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	result := &statuspb.Status{Code: int32(connectErr.Code()), Message: connectErr.Message()}
	for _, detail := range connectErr.Details() {
		value, e := detail.Value()
		if e == nil {
			packed, e := anypb.New(value)
			if e == nil {
				result.Details = append(result.Details, packed)
			}
		}
	}
	return result
}
