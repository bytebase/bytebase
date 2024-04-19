package v1

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
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
	switch r := request.(type) {
	case *v1pb.QueryRequest:
		return r.Name
	default:
		return ""
	}
}

func getRequestString(request any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		switch r := request.(type) {
		case *v1pb.QueryRequest:
			return r
		case *v1pb.CreateUserRequest:
			return r
		default:
			return nil
		}
	}()
	if m == nil {
		return "{}", nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getResponseString(response any) (string, error) {
	m := func() protoreflect.ProtoMessage {
		switch r := response.(type) {
		case *v1pb.QueryResponse:
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
		return "{}", nil
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
		v1pb.SQLService_Query_FullMethodName:
		return true
	default:
		return false
	}
}

func (in *AuditInterceptor) AuditInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	response, err := handler(ctx, request)

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

		user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
		if !ok {
			return errors.Errorf("user not found")
		}

		p := &storepb.AuditLog{
			Method:   serverInfo.FullMethod,
			Resource: getRequestResource(request),
			Severity: storepb.AuditLog_INFO,
			User:     common.FormatUserEmail(user.Email),
			Request:  requestString,
			Response: responseString,
		}

		if err := in.store.CreateAuditLog(ctx, p); err != nil {
			return errors.Wrapf(err, "failed to create audit log")
		}
		return nil
	}(); err != nil {
		slog.Warn("audit interceptor: failed to create audit log", log.BBError(err))
	}

	return response, err
}
