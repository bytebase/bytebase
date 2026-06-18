package server

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/telemetry"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func reportSQLReviewConfigSnapshot(ctx context.Context, stores *store.Store, workspaceID string) {
	ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID)
	users, err := stores.ListUsers(ctx, &store.FindUserMessage{})
	if err != nil {
		slog.Warn("failed to list users for SQL review config telemetry", log.BBError(err))
		return
	}
	email, emailDomains := buildSQLReviewTelemetryIdentity(users)
	if email == "" {
		slog.Warn("skipping SQL review config telemetry because no active end user was found")
		return
	}

	service := apiv1.NewReviewConfigService(stores)
	response, err := service.ListReviewConfigs(ctx, connect.NewRequest(&v1pb.ListReviewConfigsRequest{}))
	if err != nil {
		slog.Warn("failed to list SQL review configs for telemetry", log.BBError(err))
		return
	}

	snapshot, err := marshalSQLReviewConfigSnapshot(response.Msg)
	if err != nil {
		slog.Warn("failed to marshal SQL review config snapshot for telemetry", log.BBError(err))
		return
	}
	telemetry.ReportSQLReviewConfigSnapshot(ctx, workspaceID, email, snapshot, emailDomains)
}

func marshalSQLReviewConfigSnapshot(response *v1pb.ListReviewConfigsResponse) (string, error) {
	snapshot, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(snapshot), nil
}

func buildSQLReviewTelemetryIdentity(users []*store.UserMessage) (string, []string) {
	email := ""
	minID := 0
	domainSet := make(map[string]bool)
	for _, user := range users {
		if user == nil || user.Email == "" {
			continue
		}
		if email == "" || user.ID < minID {
			email = user.Email
			minID = user.ID
		}

		if i := strings.LastIndex(user.Email, "@"); i >= 0 && i+1 < len(user.Email) {
			domainSet[strings.ToLower(user.Email[i+1:])] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}
	slices.Sort(domains)
	return email, domains
}
