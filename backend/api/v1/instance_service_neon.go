package v1

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/neon"
)

func (s *InstanceService) DiscoverNeonBranches(ctx context.Context, req *connect.Request[v1pb.DiscoverNeonBranchesRequest]) (*connect.Response[v1pb.DiscoverNeonBranchesResponse], error) {
	var violations []*errdetails.BadRequest_FieldViolation
	if err := validateNeonAPIKey(req.Msg.ApiKey); err != nil {
		violations = append(violations, fieldViolation("api_key", err.Error()))
	}
	if req.Msg.ProjectId != "" && !neon.ValidResourceID(req.Msg.ProjectId) {
		violations = append(violations, fieldViolation("project_id", "must be a valid Neon project ID"))
	}
	size := int(req.Msg.PageSize)
	if size < 0 {
		violations = append(violations, fieldViolation("page_size", "must not be negative"))
	}
	if len(violations) > 0 {
		return nil, newBadRequestError(violations...)
	}
	if _, err := s.getProjectInstanceParent(ctx, req.Msg.Parent); err != nil {
		return nil, importDestinationError(err)
	}
	if size == 0 {
		size = 50
	}
	if size > 100 {
		size = 100
	}
	cursor, err := decodeNeonPageToken(s.secret, req.Msg.PageToken, common.GetWorkspaceIDFromContext(ctx), req.Msg.GetParent(), req.Msg.ProjectId)
	if err != nil {
		return nil, neonVendorError(neon.ErrInvalidPageToken)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	page, err := neon.NewClient(req.Msg.ApiKey).DiscoverPage(ctx, req.Msg.ProjectId, size, cursor)
	if err != nil {
		return nil, neonVendorError(err)
	}
	response := &v1pb.DiscoverNeonBranchesResponse{}
	for _, branch := range page.Branches {
		item := &v1pb.NeonBranch{ProjectId: branch.ProjectID, ProjectTitle: branch.ProjectName, BranchId: branch.ID, BranchTitle: branch.Name}
		for _, database := range branch.Databases {
			item.Databases = append(item.Databases, &v1pb.NeonDatabase{Name: database.Name, Owner: database.OwnerName})
		}
		for _, role := range branch.Roles {
			item.Roles = append(item.Roles, role.Name)
		}
		if branch.Error != nil {
			item.DiscoveryError = importStatus(neonVendorError(branch.Error))
		}
		response.Branches = append(response.Branches, item)
	}
	for _, failure := range page.Failures {
		response.Failures = append(response.Failures, &v1pb.NeonProjectDiscoveryFailure{ProjectId: failure.ProjectID, Status: importStatus(neonVendorError(failure.Error))})
	}
	if page.NextPageToken != "" {
		token, err := encodeNeonPageToken(s.secret, page.NextPageToken, common.GetWorkspaceIDFromContext(ctx), req.Msg.GetParent(), req.Msg.ProjectId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("could not encode discovery cursor"))
		}
		response.NextPageToken = token
	}
	return connect.NewResponse(response), nil
}

// neonImportResolver resolves a Neon branch source to a PostgreSQL instance
// on the branch's direct read-write endpoint.
func neonImportResolver(client *neon.Client) func(context.Context, *v1pb.InstanceImport) (*importTarget, error) {
	return func(ctx context.Context, entry *v1pb.InstanceImport) (*importTarget, error) {
		branch := entry.GetNeonBranch()
		connection, err := client.Resolve(ctx, branch.GetProjectId(), branch.GetBranchId(), entry.Username, entry.ConnectionDatabase, entry.Password)
		if err != nil {
			return nil, neonVendorError(err)
		}
		return &importTarget{
			engine:     v1pb.Engine_POSTGRES,
			vendor:     v1pb.Vendor_NEON,
			dataSource: &v1pb.DataSource{Id: "admin", Type: v1pb.DataSourceType_ADMIN, Username: connection.Username, Password: connection.Password, Host: connection.Host, Port: connection.Port, Database: connection.Database, UseSsl: true, VerifyTlsCertificate: true},
		}, nil
	}
}

func validateNeonAPIKey(key string) error {
	if strings.TrimSpace(key) == "" || len(key) > 8192 || strings.ContainsAny(key, "\r\n") {
		return errors.New("enter a valid Neon API key")
	}
	return nil
}

// neonVendorError maps a Neon client error to the import error model. Neon's
// 403 and 404 stay off PERMISSION_DENIED and NOT_FOUND, which on these RPCs
// mean the caller's Bytebase permission and the Bytebase destination.
func neonVendorError(err error) *connect.Error {
	vendorError := func(code connect.Code, reason, message string) *connect.Error {
		return newVendorError(code, reason, message, v1pb.Vendor_NEON)
	}
	switch {
	case errors.Is(err, neon.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, errors.New("check the Neon project, branch, and role identifiers"))
	case errors.Is(err, neon.ErrInvalidPageToken):
		return newReasonError(connect.CodeInvalidArgument, reasonInvalidPageToken, "Invalid discovery page token", nil)
	case errors.Is(err, neon.ErrDatabaseNotFound):
		return vendorError(connect.CodeFailedPrecondition, reasonVendorDatabaseNotFound, "The selected database no longer exists on this branch. Discover branches again.")
	case errors.Is(err, neon.ErrRoleNotFound):
		return vendorError(connect.CodeFailedPrecondition, reasonVendorUserNotFound, "The selected role no longer exists on this branch. Discover branches again.")
	case errors.Is(err, neon.ErrEndpointUnavailable):
		return vendorError(connect.CodeFailedPrecondition, reasonVendorEndpointUnavailable, "A direct read-write endpoint is required")
	case errors.Is(err, neon.ErrPasswordRequired):
		return vendorError(connect.CodeFailedPrecondition, reasonVendorPasswordRequired, "Enter the database role password")
	case errors.Is(err, context.Canceled):
		return requestInterruptedError(err)
	case errors.Is(err, context.DeadlineExceeded):
		return vendorError(connect.CodeDeadlineExceeded, reasonVendorUnavailable, "Neon did not answer before the request deadline")
	case errors.Is(err, neon.ErrUnavailable):
		return vendorError(connect.CodeUnavailable, reasonVendorUnavailable, "Neon is temporarily unavailable")
	}
	var api *neon.APIError
	if errors.As(err, &api) {
		switch {
		case api.StatusCode == 401:
			return vendorError(connect.CodeFailedPrecondition, reasonVendorCredentialInvalid, "The Neon API key is invalid or expired")
		case api.StatusCode == 403:
			return vendorError(connect.CodeFailedPrecondition, reasonVendorAccessDenied, "The Neon API key cannot access this resource")
		case api.StatusCode == 404:
			return vendorError(connect.CodeFailedPrecondition, reasonVendorResourceNotFound, "The Neon resource was not found")
		case api.StatusCode == 412:
			return vendorError(connect.CodeFailedPrecondition, reasonVendorPasswordRequired, "Enter the database role password")
		case api.StatusCode == 429:
			result := vendorError(connect.CodeResourceExhausted, reasonVendorRateLimited, "Neon rate limit exceeded")
			if api.RetryAfter > 0 {
				if detail, e := connect.NewErrorDetail(&errdetails.RetryInfo{RetryDelay: durationpb.New(api.RetryAfter)}); e == nil {
					result.AddDetail(detail)
				}
			}
			return result
		case api.StatusCode >= 500:
			return vendorError(connect.CodeUnavailable, reasonVendorUnavailable, "Neon is temporarily unavailable")
		}
	}
	return vendorError(connect.CodeInternal, reasonVendorResponseInvalid, "Neon returned a response Bytebase does not understand")
}
