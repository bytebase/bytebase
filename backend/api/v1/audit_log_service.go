package v1

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/export"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

type AuditLogService struct {
	v1connect.UnimplementedAuditLogServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
}

func NewAuditLogService(store *store.Store, licenseService *enterprise.LicenseService) *AuditLogService {
	return &AuditLogService{
		store:          store,
		licenseService: licenseService,
	}
}

func (s *AuditLogService) SearchAuditLogs(ctx context.Context, request *connect.Request[v1pb.SearchAuditLogsRequest]) (*connect.Response[v1pb.SearchAuditLogsResponse], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	filterQ, err := store.GetSearchAuditLogsFilter(request.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Apply retention-based filtering based on the plan
	retentionCutoff := s.licenseService.GetAuditLogRetentionCutoff()
	if retentionCutoff != nil {
		filterQ = store.ApplyRetentionFilter(filterQ, retentionCutoff)
	}

	orderByKeys, err := store.GetAuditLogOrders(request.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
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
	if request.Msg.Parent != "" && request.Msg.Parent != "projects/-" {
		project = &request.Msg.Parent
	}
	auditLogFind := &store.AuditLogFind{
		Project:     project,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		FilterQ:     filterQ,
		OrderByKeys: orderByKeys,
	}
	auditLogs, err := s.store.SearchAuditLogs(ctx, auditLogFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var nextPageToken string
	if len(auditLogs) == limitPlusOne {
		auditLogs = auditLogs[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	v1AuditLogs := convertToAuditLogs(auditLogs)
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
		Parent:    request.Msg.Parent,
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
		if content, err = export.CSV(result); err != nil {
			return nil, err
		}
	case v1pb.ExportFormat_JSON:
		if content, err = export.JSON(result); err != nil {
			return nil, err
		}
	case v1pb.ExportFormat_XLSX:
		if content, err = export.XLSX(result); err != nil {
			return nil, err
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported export format: %s", request.Msg.Format.String()))
	}

	return connect.NewResponse(&v1pb.ExportAuditLogsResponse{Content: content, NextPageToken: searchAuditLogsResult.Msg.NextPageToken}), nil
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
		Name:        fmt.Sprintf("%s/%s%d", l.Payload.Parent, common.AuditLogPrefix, l.ID),
		CreateTime:  timestamppb.New(l.CreatedAt),
		User:        l.Payload.User,
		Method:      l.Payload.Method,
		Severity:    convertToAuditLogSeverity(l.Payload.Severity),
		Resource:    l.Payload.Resource,
		Request:     l.Payload.Request,
		Response:    l.Payload.Response,
		Status:      l.Payload.Status,
		Latency:     l.Payload.Latency,
		ServiceData: l.Payload.ServiceData,
	}
}

func convertToAuditLogSeverity(s storepb.AuditLog_Severity) v1pb.AuditLog_Severity {
	switch s {
	case storepb.AuditLog_SEVERITY_UNSPECIFIED:
		return v1pb.AuditLog_SEVERITY_UNSPECIFIED
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
		return v1pb.AuditLog_SEVERITY_UNSPECIFIED
	}
}
