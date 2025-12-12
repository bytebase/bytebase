package v1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func convertToV1Advice(advice *storepb.Advice) *v1pb.Advice {
	return &v1pb.Advice{
		Status:        convertAdviceStatus(advice.Status),
		Code:          int32(advice.Code),
		Title:         advice.Title,
		Content:       advice.Content,
		StartPosition: convertToPosition(advice.StartPosition),
		EndPosition:   convertToPosition(advice.EndPosition),
	}
}

func convertAdviceStatus(status storepb.Advice_Status) v1pb.Advice_Level {
	switch status {
	case storepb.Advice_SUCCESS:
		return v1pb.Advice_SUCCESS
	case storepb.Advice_WARNING:
		return v1pb.Advice_WARNING
	case storepb.Advice_ERROR:
		return v1pb.Advice_ERROR
	default:
		return v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED
	}
}

func (s *SQLService) convertToV1QueryHistory(ctx context.Context, history *store.QueryHistoryMessage) (*v1pb.QueryHistory, error) {
	creator, err := s.store.GetUserByEmail(ctx, history.Creator)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, errors.Errorf("cannot found user with email %s", history.Creator)
	}

	historyType := v1pb.QueryHistory_TYPE_UNSPECIFIED
	switch history.Type {
	case store.QueryHistoryTypeExport:
		historyType = v1pb.QueryHistory_EXPORT
	case store.QueryHistoryTypeQuery:
		historyType = v1pb.QueryHistory_QUERY
	default:
	}

	return &v1pb.QueryHistory{
		Name:       fmt.Sprintf("queryHistories/%d", history.UID),
		Statement:  history.Statement,
		Error:      history.Payload.Error,
		Database:   history.Database,
		Creator:    common.FormatUserEmail(creator.Email),
		CreateTime: timestamppb.New(history.CreatedAt),
		Duration:   history.Payload.Duration,
		Type:       historyType,
	}, nil
}
