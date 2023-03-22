package store

import (
	"context"

	v1alpha1 "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type RiskMessage struct {
	Source     string
	level      int64
	name       string
	active     bool
	expression *v1alpha1.ParsedExpr

	// Output only
	ID int64
}

func (*Store) ListRisks(ctx context.Context) ([]*RiskMessage, error) {
	return nil, nil
}

func (*Store) CreateRisk(ctx context.Context, risk *RiskMessage, creatorID int) error {
	return nil
}
