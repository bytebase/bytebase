package v1

import (
	v1pb "github.com/bytebase/bytebase/proto/generated-go/api/v1alpha"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
