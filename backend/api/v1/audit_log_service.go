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
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type AuditLogService struct {
	v1pb.UnimplementedAuditLogServiceServer
	store          *store.Store
	iamManager     *iam.Manager
	licenseService enterprise.LicenseService
}

func NewAuditLogService(store *store.Store, iamManager *iam.Manager, licenseService enterprise.LicenseService) *AuditLogService {
	return &AuditLogService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

func (s *AuditLogService) SearchAuditLogs(ctx context.Context, request *v1pb.SearchAuditLogsRequest) (*v1pb.SearchAuditLogsResponse, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureAuditLog); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	filter, serr := getSearchAuditLogsFilter(request.Filter)
	if serr != nil {
		return nil, serr.Err()
	}

	orderByKeys, err := getSearchAuditLogsOrderByKeys(request.OrderBy)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get order by keys, error: %v", err)
	}

	limit, offset, err := parseLimitAndOffset(request.PageToken, int(request.PageSize))
	if err != nil {
		return nil, err
	}
	limitPlusOne := limit + 1

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	permissionFilter, serr := getSearchAuditLogsPermissionFilter(ctx, s.store, user, s.iamManager)
	if serr != nil {
		return nil, serr.Err()
	}

	auditLogFind := &store.AuditLogFind{
		Limit:            &limitPlusOne,
		Offset:           &offset,
		Filter:           filter,
		OrderByKeys:      orderByKeys,
		PermissionFilter: permissionFilter,
	}
	auditLogs, err := s.store.SearchAuditLogs(ctx, auditLogFind)
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

	v1AuditLogs, err := convertToAuditLogs(ctx, s.store, auditLogs)
	if err != nil {
		return nil, err
	}
	return &v1pb.SearchAuditLogsResponse{
		AuditLogs:     v1AuditLogs,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *AuditLogService) ExportAuditLogs(ctx context.Context, request *v1pb.ExportAuditLogsRequest) (*v1pb.ExportAuditLogsResponse, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureAuditLog); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	searchAuditLogsResult, err := s.SearchAuditLogs(ctx, &v1pb.SearchAuditLogsRequest{
		Filter:  request.Filter,
		OrderBy: request.OrderBy,
		// Default 1000 rows to avoid OOM for now.
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}

	result := &v1pb.QueryResult{
		ColumnNames: []string{"time", "user", "method", "severity", "resource", "request", "response", "status"},
	}
	for _, auditLog := range searchAuditLogsResult.AuditLogs {
		queryRow := &v1pb.QueryRow{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.CreateTime.AsTime().Format(time.RFC3339)}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.User}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Method}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Severity.String()}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Resource}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Request}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Response}},
		}}
		if auditLog.Status != nil {
			queryRow.Values = append(queryRow.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: auditLog.Status.String()}})
		} else {
			queryRow.Values = append(queryRow.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}})
		}
		result.Rows = append(result.Rows, queryRow)
	}

	var content []byte
	switch request.Format {
	case v1pb.ExportFormat_CSV:
		if content, err = exportCSV(result); err != nil {
			return nil, err
		}
	case v1pb.ExportFormat_JSON:
		if content, err = exportJSON(result); err != nil {
			return nil, err
		}
	case v1pb.ExportFormat_XLSX:
		if content, err = exportXLSX(result); err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported export format: %s", request.Format.String())
	}

	return &v1pb.ExportAuditLogsResponse{Content: content}, nil
}

func convertToAuditLogs(ctx context.Context, stores *store.Store, auditLogs []*store.AuditLog) ([]*v1pb.AuditLog, error) {
	var ls []*v1pb.AuditLog
	for _, log := range auditLogs {
		l, err := convertToAuditLog(ctx, stores, log)
		if err != nil {
			return nil, err
		}
		ls = append(ls, l)
	}
	return ls, nil
}

func convertToAuditLog(ctx context.Context, stores *store.Store, l *store.AuditLog) (*v1pb.AuditLog, error) {
	var user string
	if l.Payload.User != "" {
		uid, err := common.GetUserID(l.Payload.User)
		if err != nil {
			return nil, err
		}
		u, err := stores.GetUserByID(ctx, uid)
		if err != nil {
			return nil, err
		}
		user = common.FormatUserEmail(u.Email)
	}
	return &v1pb.AuditLog{
		Name:        fmt.Sprintf("%s/%s%d", l.Payload.Parent, common.AuditLogPrefix, l.ID),
		CreateTime:  timestamppb.New(time.Unix(l.CreatedTs, 0)),
		User:        user,
		Method:      l.Payload.Method,
		Severity:    convertToAuditLogSeverity(l.Payload.Severity),
		Resource:    l.Payload.Resource,
		Request:     l.Payload.Request,
		Response:    l.Payload.Response,
		Status:      l.Payload.Status,
		ServiceData: l.Payload.ServiceData,
	}, nil
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
			functionName := expr.AsCall().FunctionName()
			switch functionName {
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
				case "resource", "parent", "method", "user", "severity":
				default:
					return "", errors.Errorf("unknown variable %s", variable)
				}
				positionalArgs = append(positionalArgs, value)
				return fmt.Sprintf("payload->>'%s'=$%d", variable, len(positionalArgs)), nil

			case "_>=_", "_<=_":
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
				if variable != "create_time" {
					return "", errors.Errorf(`">=" and "<=" are only supported for "create_time"`)
				}

				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return "", errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				ts := t.Unix()
				positionalArgs = append(positionalArgs, ts)
				if functionName == "_>=_" {
					return fmt.Sprintf("created_ts >= $%d", len(positionalArgs)), nil
				}
				return fmt.Sprintf("created_ts <= $%d", len(positionalArgs)), nil

			default:
				return "", errors.Errorf("unexpected function %v", functionName)
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
		Where: "(" + where + ")",
	}, nil
}

func getSearchAuditLogsOrderByKeys(orderBy string) ([]store.OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	orderByKeys := []store.OrderByKey{}
	for _, orderByKey := range keys {
		key := ""
		if orderByKey.key == "create_time" {
			key = "created_ts"
		}
		if key == "" {
			return nil, status.Errorf(codes.InvalidArgument, "invalid order by key %v", orderByKey.key)
		}

		sortOrder := store.ASC
		if !orderByKey.isAscend {
			sortOrder = store.DESC
		}
		orderByKeys = append(orderByKeys, store.OrderByKey{
			Key:       key,
			SortOrder: sortOrder,
		})
	}
	return orderByKeys, nil
}

func getSearchAuditLogsPermissionFilter(ctx context.Context, s *store.Store, user *store.UserMessage, iamManager *iam.Manager) (*store.AuditLogPermissionFilter, *status.Status) {
	projectIDs, err := getProjectIDsWithPermission(ctx, s, user, iamManager, iam.PermissionAuditLogsGet)
	if err != nil {
		return nil, status.Newf(codes.Internal, "failed to get projectIDs with permission")
	}
	if projectIDs == nil {
		return nil, nil
	}

	var projects []string
	for _, p := range *projectIDs {
		projects = append(projects, common.FormatProject(p))
	}
	return &store.AuditLogPermissionFilter{
		Projects: projects,
	}, nil
}
