package v1

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func convertToPosition(position *storepb.Position) *v1pb.Position {
	if position == nil {
		return nil
	}

	return &v1pb.Position{
		Line:   position.Line,
		Column: position.Column,
	}
}
