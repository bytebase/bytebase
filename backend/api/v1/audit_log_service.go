package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type AuditLogService struct {
	v1pb.UnimplementedAuditLogServiceServer
	store *store.Store
}

func NewAuditLogService(store *store.Store) *AuditLogService {
	return &AuditLogService{
		store: store,
	}
}

func (s *AuditLogService) SearchAuditLogs(ctx context.Context, request *v1pb.SearchAuditLogsRequest) (*v1pb.SearchAuditLogsResponse, error) {
	filter, serr := getSearchAuditLogsFilter(request.Filter)
	if serr != nil {
		return nil, serr.Err()
	}

	limit, offset, err := parseLimitAndOffset(request.PageToken, int(request.PageSize))
	if err != nil {
		return nil, err
	}
	limitPlusOne := limit + 1

	auditLogs, err := s.store.SearchAuditLogs(ctx, &store.AuditLogFind{
		Filter: filter,
		Limit:  &limitPlusOne,
		Offset: &offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get audit logs, error: %v", err)
	}

	var nextPageToken string
	if len(auditLogs) == limitPlusOne {
		auditLogs = auditLogs[:limit]
		token, err := getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		nextPageToken = token
	}

	return &v1pb.SearchAuditLogsResponse{
		AuditLogs:     convertToAuditLogs(auditLogs),
		NextPageToken: nextPageToken,
	}, nil
}

func convertToAuditLogs(auditLogs []*store.AuditLog) []*v1pb.AuditLog {
	var ls []*v1pb.AuditLog
	for _, log := range auditLogs {
		ls = append(ls, convertToAuditLog(log))
	}
	return ls
}

func convertToAuditLog(l *store.AuditLog) *v1pb.AuditLog {
	return &v1pb.AuditLog{
		Name:       fmt.Sprintf("%s/%s%d", l.Payload.Parent, common.AuditLogPrefix, l.ID),
		CreateTime: timestamppb.New(time.Unix(l.CreatedTs, 0)),
		User:       l.Payload.User,
		Method:     l.Payload.Method,
		Severity:   convertToAuditLogSeverity(l.Payload.Severity),
		Resource:   l.Payload.Resource,
		Request:    l.Payload.Request,
		Response:   l.Payload.Response,
		Status:     l.Payload.Status,
	}
}

func convertToAuditLogSeverity(s storepb.AuditLog_Severity) v1pb.AuditLog_Severity {
	switch s {
	case storepb.AuditLog_DEFAULT:
		return v1pb.AuditLog_DEFAULT
	case storepb.AuditLog_DEBUG:
		return v1pb.AuditLog_DEBUG
	case storepb.AuditLog_INFO:
		return v1pb.AuditLog_INFO
	case storepb.AuditLog_NOTICE:
		return v1pb.AuditLog_NOTICE
	case storepb.AuditLog_WARNING:
		return v1pb.AuditLog_WARNING
	case storepb.AuditLog_ERROR:
		return v1pb.AuditLog_ERROR
	case storepb.AuditLog_CRITICAL:
		return v1pb.AuditLog_CRITICAL
	case storepb.AuditLog_ALERT:
		return v1pb.AuditLog_ALERT
	case storepb.AuditLog_EMERGENCY:
		return v1pb.AuditLog_EMERGENCY
	default:
		return v1pb.AuditLog_DEFAULT
	}
}

func getSearchAuditLogsFilter(filter string) (*store.AuditLogFilter, *status.Status) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, status.Newf(codes.Internal, "failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, status.Newf(codes.InvalidArgument, "failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			switch expr.AsCall().FunctionName() {
			case "_||_":
				var args []string
				for _, arg := range expr.AsCall().Args() {
					s, err := getFilter(arg)
					if err != nil {
						return "", err
					}
					args = append(args, "("+s+")")
				}
				return strings.Join(args, " OR "), nil

			case "_&&_":
				var args []string
				for _, arg := range expr.AsCall().Args() {
					s, err := getFilter(arg)
					if err != nil {
						return "", err
					}
					args = append(args, "("+s+")")
				}
				return strings.Join(args, " AND "), nil

			case "_==_":
				var variable, value string
				for _, arg := range expr.AsCall().Args() {
					switch arg.Kind() {
					case celast.IdentKind:
						variable = arg.AsIdent()
					case celast.LiteralKind:
						lit, ok := arg.AsLiteral().Value().(string)
						if !ok {
							return "", errors.Errorf("expect string, got %T, hint: filter literals should be string", arg.AsLiteral().Value())
						}
						value = lit
					}
				}
				switch variable {
				case "resource", "parent", "method", "user":
				default:
					return "", errors.Errorf("unknown variable %s", variable)
				}
				positionalArgs = append(positionalArgs, value)
				return fmt.Sprintf("payload->>'%s'=$%d", variable, len(positionalArgs)), nil

			default:
				return "", errors.Errorf("unexpected function %v", expr.AsCall().FunctionName())
			}

		default:
			return "", errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, status.Newf(codes.InvalidArgument, "failed to get filter, error: %v", err)
	}

	return &store.AuditLogFilter{
		Args:  positionalArgs,
		Where: where,
	}, nil
}
