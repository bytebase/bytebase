package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

type AuditLogService struct {
	v1connect.UnimplementedAuditLogServiceHandler
	store          *store.Store
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
}

func NewAuditLogService(store *store.Store, iamManager *iam.Manager, licenseService *enterprise.LicenseService) *AuditLogService {
	return &AuditLogService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

func (s *AuditLogService) SearchAuditLogs(ctx context.Context, request *connect.Request[v1pb.SearchAuditLogsRequest]) (*connect.Response[v1pb.SearchAuditLogsResponse], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	filter, err := s.getSearchAuditLogsFilter(ctx, request.Msg.Filter)
	if err != nil {
		return nil, err
	}

	// Apply retention-based filtering based on the plan
	retentionCutoff := s.licenseService.GetAuditLogRetentionCutoff()
	if retentionCutoff != nil {
		filter = s.applyRetentionFilter(filter, retentionCutoff)
	}

	orderByKeys, err := getSearchAuditLogsOrderByKeys(request.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get order by keys"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.Msg.PageToken,
		limit:   int(request.Msg.PageSize),
		maximum: 5000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	var project *string
	if request.Msg.Parent != "" {
		project = &request.Msg.Parent
	}
	auditLogFind := &store.AuditLogFind{
		Project:     project,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		Filter:      filter,
		OrderByKeys: orderByKeys,
	}
	auditLogs, err := s.store.SearchAuditLogs(ctx, auditLogFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get audit logs"))
	}

	var nextPageToken string
	if len(auditLogs) == limitPlusOne {
		auditLogs = auditLogs[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
	}

	v1AuditLogs, err := convertToAuditLogs(ctx, s.store, auditLogs)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1pb.SearchAuditLogsResponse{
		AuditLogs:     v1AuditLogs,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *AuditLogService) ExportAuditLogs(ctx context.Context, request *connect.Request[v1pb.ExportAuditLogsRequest]) (*connect.Response[v1pb.ExportAuditLogsResponse], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if request.Msg.Filter == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("filter is required to export audit logs"))
	}
	searchAuditLogsResult, err := s.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Filter:    request.Msg.Filter,
		OrderBy:   request.Msg.OrderBy,
		PageSize:  request.Msg.PageSize,
		PageToken: request.Msg.PageToken,
	}))
	if err != nil {
		return nil, err
	}

	result := &v1pb.QueryResult{
		ColumnNames: []string{"time", "user", "method", "severity", "resource", "request", "response", "status"},
	}
	for _, auditLog := range searchAuditLogsResult.Msg.AuditLogs {
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
	switch request.Msg.Format {
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
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported export format: %s", request.Msg.Format.String()))
	}

	return connect.NewResponse(&v1pb.ExportAuditLogsResponse{Content: content, NextPageToken: searchAuditLogsResult.Msg.NextPageToken}), nil
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
		CreateTime:  timestamppb.New(l.CreatedAt),
		User:        user,
		Method:      l.Payload.Method,
		Severity:    convertToAuditLogSeverity(l.Payload.Severity),
		Resource:    l.Payload.Resource,
		Request:     l.Payload.Request,
		Response:    l.Payload.Response,
		Status:      l.Payload.Status,
		Latency:     l.Payload.Latency,
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

func (s *AuditLogService) getSearchAuditLogsFilter(ctx context.Context, filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return "", errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				switch variable {
				case "resource", "method", "user", "severity":
				default:
					return "", errors.Errorf("unknown variable %s", variable)
				}
				if variable == "user" {
					// Convert user email to user id.
					// e.g. users/y@bb.com -> users/101
					user, err := s.getUserByIdentifier(ctx, value)
					if err != nil {
						return "", err
					}
					value = fmt.Sprintf("%s%d", common.UserNamePrefix, user.ID)
				}
				positionalArgs = append(positionalArgs, value)
				return fmt.Sprintf("payload->>'%s'=$%d", variable, len(positionalArgs)), nil

			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return "", errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				if variable != "create_time" {
					return "", errors.Errorf(`">=" and "<=" are only supported for "create_time"`)
				}

				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return "", errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				positionalArgs = append(positionalArgs, t)
				if functionName == celoperators.GreaterEquals {
					return fmt.Sprintf("created_at >= $%d", len(positionalArgs)), nil
				}
				return fmt.Sprintf("created_at <= $%d", len(positionalArgs)), nil

			default:
				return "", errors.Errorf("unexpected function %v", functionName)
			}

		default:
			return "", errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get filter"))
	}

	return &store.ListResourceFilter{
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
			key = "created_at"
		}
		if key == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid order by key %v", orderByKey.key))
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

func (s *AuditLogService) getUserByIdentifier(ctx context.Context, identifier string) (*store.UserMessage, error) {
	email := strings.TrimPrefix(identifier, "users/")
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid empty creator identifier"))
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, `failed to find user "%s"`, email))
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user %s", email)
	}
	return user, nil
}

// applyRetentionFilter merges retention-based filtering with user-provided filters.
func (*AuditLogService) applyRetentionFilter(userFilter *store.ListResourceFilter, cutoff *time.Time) *store.ListResourceFilter {
	if cutoff == nil {
		return userFilter
	}

	if userFilter == nil {
		return &store.ListResourceFilter{
			Where: "(created_at >= $1)",
			Args:  []any{*cutoff},
		}
	}

	// Combine with existing filter using AND
	return &store.ListResourceFilter{
		Where: fmt.Sprintf("(%s) AND (created_at >= $%d)", userFilter.Where, len(userFilter.Args)+1),
		Args:  append(userFilter.Args, *cutoff),
	}
}
