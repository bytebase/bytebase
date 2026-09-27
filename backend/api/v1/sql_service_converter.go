package v1

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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
