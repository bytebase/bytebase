package v1

import (
	"context"
	"log/slog"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ACLInterceptor is the v1 ACL interceptor for gRPC server.
type AuditInterceptor struct {
	store *store.Store
}

// NewAuditInterceptor returns a new v1 API ACL interceptor.
func NewAuditInterceptor(store *store.Store) *AuditInterceptor {
	return &AuditInterceptor{
		store: store,
	}
}

func (in *AuditInterceptor) AuditInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	response, rerr := handler(ctx, request)

	if err := func() error {
		if !isAuditMethod(serverInfo.FullMethod) {
			return nil
		}
		requestString, err := getRequestString(request)
		if err != nil {
			return errors.Wrapf(err, "failed to get request string")
		}
		responseString, err := getResponseString(response)
		if err != nil {
			return errors.Wrapf(err, "failed to get response string")
		}

		var user string
		if u, ok := ctx.Value(common.UserContextKey).(*store.UserMessage); ok {
			user = common.FormatUserEmail(u.Email)
		}

		st, _ := status.FromError(rerr)

		projectIDs, ok := common.GetProjectIDsFromContext(ctx)
		if !ok {
			return errors.Errorf("failed to get projects ids from context")
		}
		var parents []string
		if len(projectIDs) == 0 {
			workspaceID, err := in.store.GetWorkspaceID(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to get workspace id")
			}
			parents = append(parents, common.FormatWorkspace(workspaceID))
		} else {
			for _, projectID := range projectIDs {
				parents = append(parents, common.FormatProject(projectID))
			}
		}

		createAuditLogCtx := context.WithoutCancel(ctx)
		for _, parent := range parents {
			p := &storepb.AuditLog{
				Parent:   parent,
				Method:   serverInfo.FullMethod,
				Resource: getRequestResource(request),
				Severity: storepb.AuditLog_INFO,
				User:     user,
				Request:  requestString,
				Response: responseString,
				Status:   st.Proto(),
			}
			if err := in.store.CreateAuditLog(createAuditLogCtx, p); err != nil {
				return errors.Wrapf(err, "failed to create audit log")
			}
		}

		return nil
	}(); err != nil {
		slog.Warn("audit interceptor: failed to create audit log", log.BBError(err))
	}

	return response, rerr
}

func getRequestResource(request any) string {
	if request == nil || reflect.ValueOf(request).IsNil() {
		return ""
	}
	switch r := request.(type) {
	case *v1pb.QueryRequest:
		return r.Name
	case *v1pb.ExportRequest:
		return r.Name
	case *v1pb.UpdateDatabaseRequest:
		return r.Database.Name
	case *v1pb.BatchUpdateDatabasesRequest:
		return r.Parent
	case *v1pb.SetIamPolicyRequest:
		return r.Project
	case *v1pb.CreateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.UpdateUserRequest:
		return r.GetUser().GetName()
	default:
		return ""
	}
}

func getRequestString(request any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		if request == nil || reflect.ValueOf(request).IsNil() {
			return nil
		}
		switch r := request.(type) {
		case *v1pb.QueryRequest:
			return r
		case *v1pb.ExportRequest:
			//nolint:revive
			r = proto.Clone(r).(*v1pb.ExportRequest)
			r.Password = ""
			return r
		case *v1pb.UpdateDatabaseRequest:
			return r
		case *v1pb.BatchUpdateDatabasesRequest:
			return r
		case *v1pb.SetIamPolicyRequest:
			return r
		case *v1pb.CreateUserRequest:
			return redactCreateUserRequest(r)
		case *v1pb.UpdateUserRequest:
			return redactUpdateUserRequest(r)
		default:
			return nil
		}
	}()
	if m == nil {
		return "", nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getResponseString(response any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		if response == nil || reflect.ValueOf(response).IsNil() {
			return nil
		}
		switch r := response.(type) {
		case *v1pb.QueryResponse:
			return redactQueryResponse(r)
		case *v1pb.ExportResponse:
			return nil
		case *v1pb.Database:
			return r
		case *v1pb.BatchUpdateDatabasesResponse:
			return r
		case *v1pb.IamPolicy:
			return r
		case *v1pb.User:
			return redactUser(r)
		default:
			return nil
		}
	}()
	if m == nil {
		return "", nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func redactCreateUserRequest(r *v1pb.CreateUserRequest) *v1pb.CreateUserRequest {
	if r == nil {
		return nil
	}
	return &v1pb.CreateUserRequest{
		User: redactUser(r.User),
	}
}

func redactUpdateUserRequest(r *v1pb.UpdateUserRequest) *v1pb.UpdateUserRequest {
	if r == nil {
		return nil
	}
	return &v1pb.UpdateUserRequest{
		User:                    redactUser(r.User),
		UpdateMask:              r.UpdateMask,
		OtpCode:                 r.OtpCode,
		RegenerateTempMfaSecret: r.RegenerateTempMfaSecret,
		RegenerateRecoveryCodes: r.RegenerateRecoveryCodes,
	}
}

func redactUser(r *v1pb.User) *v1pb.User {
	if r == nil {
		return nil
	}
	return &v1pb.User{
		Name:     r.Name,
		Email:    r.Email,
		Title:    r.Title,
		UserType: r.UserType,
	}
}

func redactQueryResponse(r *v1pb.QueryResponse) *v1pb.QueryResponse {
	if r == nil {
		return nil
	}
	n := &v1pb.QueryResponse{
		Results:     nil,
		Advices:     nil,
		AllowExport: r.AllowExport,
	}
	for _, result := range r.Results {
		n.Results = append(n.Results, &v1pb.QueryResult{
			ColumnNames:     result.ColumnNames,
			ColumnTypeNames: result.ColumnTypeNames,
			Rows:            nil, // Redacted
			Masked:          result.Masked,
			Sensitive:       result.Sensitive,
			Error:           result.Error,
			Latency:         result.Latency,
			Statement:       result.Statement,
		})
	}
	return n
}

func isAuditMethod(method string) bool {
	switch method {
	case
		v1pb.AuthService_CreateUser_FullMethodName,
		v1pb.AuthService_UpdateUser_FullMethodName,
		v1pb.DatabaseService_UpdateDatabase_FullMethodName,
		v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName,
		v1pb.ProjectService_SetIamPolicy_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.SQLService_Query_FullMethodName:
		return true
	default:
		return false
	}
}
