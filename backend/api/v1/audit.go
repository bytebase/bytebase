package v1

import (
	"context"
	"log/slog"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// used for replacing sensitive fields.
var (
	maskedString string
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

func createAuditLog(ctx context.Context, request, response any, method string, storage *store.Store, serviceData *anypb.Any, rerr error) error {
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
		user = common.FormatUserUID(u.ID)
	}

	st, _ := status.FromError(rerr)

	authContextAny := ctx.Value(common.AuthContextKey)
	authContext, ok := authContextAny.(*common.AuthContext)
	if !ok {
		return status.Errorf(codes.Internal, "auth context not found")
	}

	var parents []string
	if authContext.HasWorkspaceResource() {
		workspaceID, err := storage.GetWorkspaceID(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to get workspace id")
		}
		parents = append(parents, common.FormatWorkspace(workspaceID))
	} else {
		for _, projectID := range authContext.GetProjectResources() {
			parents = append(parents, common.FormatProject(projectID))
		}
	}

	createAuditLogCtx := context.WithoutCancel(ctx)
	for _, parent := range parents {
		p := &storepb.AuditLog{
			Parent:      parent,
			Method:      method,
			Resource:    getRequestResource(request),
			Severity:    storepb.AuditLog_INFO,
			User:        user,
			Request:     requestString,
			Response:    responseString,
			Status:      st.Proto(),
			ServiceData: serviceData,
		}
		if err := storage.CreateAuditLog(createAuditLogCtx, p); err != nil {
			return errors.Wrapf(err, "failed to create audit log")
		}
	}

	return nil
}

func (in *AuditInterceptor) AuditInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	var serviceData *anypb.Any
	ctx = common.WithSetServiceData(ctx, func(a *anypb.Any) {
		serviceData = a
	})
	response, rerr := handler(ctx, request)
	if isAuditMethod(serverInfo.FullMethod) {
		if err := func() error {
			return createAuditLog(ctx, request, response, serverInfo.FullMethod, in.store, serviceData, rerr)
		}(); err != nil {
			slog.Warn("audit interceptor: failed to create audit log", log.BBError(err))
		}
	}
	return response, rerr
}

type auditStream struct {
	grpc.ServerStream
	needAudit  bool
	curRequest any
	ctx        context.Context
	method     string
	storage    *store.Store
}

func (s *auditStream) RecvMsg(request any) error {
	err := s.ServerStream.RecvMsg(request)
	if err != nil {
		return err
	}
	// audit log.
	if s.needAudit {
		s.curRequest = request
	}
	return nil
}

func (s *auditStream) SendMsg(resp any) error {
	err := s.ServerStream.SendMsg(resp)
	if err != nil {
		return err
	}
	// audit log.
	if s.needAudit && s.curRequest != nil {
		if auditErr := createAuditLog(s.ctx, s.curRequest, resp, s.method, s.storage, nil, nil); auditErr != nil {
			return auditErr
		}
	}

	return nil
}

func (in *AuditInterceptor) AuditStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	overrideStream, ok := ss.(*overrideStream)
	if !ok {
		return errors.New("type assertions failed: grpc.ServerStream -> overrideStream")
	}

	auditStream := &auditStream{
		ServerStream: overrideStream,
		needAudit:    isStreamAuditMethod(info.FullMethod),
		ctx:          overrideStream.childCtx,
		method:       info.FullMethod,
		storage:      in.store,
	}

	err := handler(srv, auditStream)
	if err != nil {
		return createAuditLog(auditStream.ctx, auditStream.curRequest, nil, auditStream.method, auditStream.storage, nil, err)
	}

	return nil
}

func getRequestResource(request any) string {
	if request == nil || reflect.ValueOf(request).IsNil() {
		return ""
	}
	switch r := request.(type) {
	case *v1pb.QueryRequest:
		return r.Name
	case *v1pb.AdminExecuteRequest:
		return r.Name
	case *v1pb.ExportRequest:
		return r.Name
	case *v1pb.UpdateDatabaseRequest:
		return r.Database.Name
	case *v1pb.BatchUpdateDatabasesRequest:
		return r.Parent
	case *v1pb.SetIamPolicyRequest:
		return r.Resource
	case *v1pb.CreateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.UpdateUserRequest:
		return r.GetUser().GetName()
	case *v1pb.LoginRequest:
		return r.GetEmail()
	case *v1pb.CreateRiskRequest:
		return r.GetRisk().GetName()
	case *v1pb.DeleteRiskRequest:
		return r.Name
	case *v1pb.UpdateRiskRequest:
		return r.GetRisk().GetName()
	case *v1pb.CreateEnvironmentRequest:
		return r.GetEnvironment().GetName()
	case *v1pb.UpdateEnvironmentRequest:
		return r.GetEnvironment().GetName()
	case *v1pb.DeleteEnvironmentRequest:
		return r.Name
	case *v1pb.UndeleteEnvironmentRequest:
		return r.Name
	case *v1pb.UpdateSettingRequest:
		return r.GetSetting().GetName()
	default:
	}
	return ""
}

func getRequestString(request any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		if request == nil || reflect.ValueOf(request).IsNil() {
			return nil
		}
		switch r := request.(type) {
		case *v1pb.ExportRequest:
			//nolint:revive
			r = proto.Clone(r).(*v1pb.ExportRequest)
			r.Password = ""
			return r
		case *v1pb.CreateUserRequest:
			return redactCreateUserRequest(r)
		case *v1pb.UpdateUserRequest:
			return redactUpdateUserRequest(r)
		case *v1pb.LoginRequest:
			if r, ok := proto.Clone(r).(*v1pb.LoginRequest); ok {
				return redactLoginRequest(r)
			}
			return nil
		default:
			if p, ok := r.(protoreflect.ProtoMessage); ok {
				return p
			}
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
		case *v1pb.AdminExecuteResponse:
			return redactAdminExecuteResponse(r)
		case *v1pb.ExportResponse:
			return nil
		case *v1pb.LoginResponse:
			return nil
		case *v1pb.User:
			return redactUser(r)
		default:
			if p, ok := r.(protoreflect.ProtoMessage); ok {
				return p
			}
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

func redactLoginRequest(r *v1pb.LoginRequest) *v1pb.LoginRequest {
	if r == nil {
		return nil
	}

	//  sensitive fields blacklist.
	if r.Password != "" {
		r.Password = maskedString
	}
	if r.OtpCode != nil {
		r.OtpCode = &maskedString
	}
	if r.RecoveryCode != nil {
		r.RecoveryCode = &maskedString
	}
	if r.MfaTempToken != nil {
		r.MfaTempToken = &maskedString
	}
	if r.IdpContext != nil {
		r.IdpContext = nil
	}
	return r
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

func redactAdminExecuteResponse(r *v1pb.AdminExecuteResponse) *v1pb.AdminExecuteResponse {
	if r == nil {
		return nil
	}
	n := &v1pb.AdminExecuteResponse{
		Results: nil,
	}
	for _, result := range r.Results {
		if result == nil {
			n.Results = append(n.Results, &v1pb.QueryResult{})
			continue
		}
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
		v1pb.AuthService_Login_FullMethodName,
		v1pb.AuthService_CreateUser_FullMethodName,
		v1pb.AuthService_UpdateUser_FullMethodName,
		v1pb.ProjectService_SetIamPolicy_FullMethodName,
		v1pb.DatabaseService_UpdateDatabase_FullMethodName,
		v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.SQLService_Query_FullMethodName,
		v1pb.RiskService_CreateRisk_FullMethodName,
		v1pb.RiskService_DeleteRisk_FullMethodName,
		v1pb.RiskService_UpdateRisk_FullMethodName,
		v1pb.EnvironmentService_CreateEnvironment_FullMethodName,
		v1pb.EnvironmentService_UpdateEnvironment_FullMethodName,
		v1pb.EnvironmentService_DeleteEnvironment_FullMethodName,
		v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName,
		v1pb.SettingService_UpdateSetting_FullMethodName:
		return true
	default:
	}
	return false
}

func isStreamAuditMethod(method string) bool {
	switch method {
	case v1pb.SQLService_AdminExecute_FullMethodName:
		return true
	default:
		return false
	}
}
