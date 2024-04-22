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

func getRequestResource(request any) string {
	if request == nil || reflect.ValueOf(request).IsNil() {
		return ""
	}
	switch r := request.(type) {
	case *v1pb.QueryRequest:
		return r.Name
	case *v1pb.ExportRequest:
		return r.Name
	case *v1pb.CreateUserRequest:
		return ""
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
		case *v1pb.CreateUserRequest:
			return r
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
			return nil
		case *v1pb.ExportResponse:
			return nil
		case *v1pb.User:
			return &v1pb.User{
				Name:     r.Name,
				Email:    r.Email,
				Title:    r.Title,
				UserType: r.UserType,
			}
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

func isAuditMethod(method string) bool {
	switch method {
	case
		v1pb.AuthService_CreateUser_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.SQLService_Query_FullMethodName:
		return true
	default:
		return false
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

		p := &storepb.AuditLog{
			Method:   serverInfo.FullMethod,
			Resource: getRequestResource(request),
			Severity: storepb.AuditLog_INFO,
			User:     user,
			Request:  requestString,
			Response: responseString,
			Status:   st.Proto(),
		}

		if err := in.store.CreateAuditLog(ctx, p); err != nil {
			return errors.Wrapf(err, "failed to create audit log")
		}
		return nil
	}(); err != nil {
		slog.Warn("audit interceptor: failed to create audit log", log.BBError(err))
	}

	return response, rerr
}
