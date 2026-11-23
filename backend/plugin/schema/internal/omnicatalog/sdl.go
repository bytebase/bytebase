package omnicatalog

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// PlanSDLDropAdvices converts currentSchema to SDL, builds an engine-specific
// migration plan against userSDLText, and collects destructive-op advices.
func PlanSDLDropAdvices[Plan any](
	engine storepb.Engine,
	userSDLText string,
	currentSchema *model.DatabaseMetadata,
	buildPlan func(sourceSDL, targetSDL string) (*Plan, error),
	collect func(*Plan) []*storepb.Advice,
) ([]*storepb.Advice, error) {
	sourceSDL, err := schema.MetadataToSDL(engine, currentSchema)
	if err != nil {
		return nil, err
	}
	plan, err := buildPlan(sourceSDL, userSDLText)
	if err != nil || plan == nil {
		return nil, err
	}
	return collect(plan), nil
}

// SDLDropAdvice builds a WARNING advice for a destructive SDL drop/alter op.
func SDLDropAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLDropOperation.Int32(),
		Title:   "DROP operation detected",
		Content: content,
	}
}

// SDLReplaceAdvice builds a WARNING advice for an in-place SDL replace op.
func SDLReplaceAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLReplaceOperation.Int32(),
		Title:   "CREATE OR REPLACE operation detected",
		Content: content,
	}
}
