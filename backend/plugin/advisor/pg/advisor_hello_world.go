package pg

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*HelloWorldAdvisor)(nil)
)

const (
	// HelloWorldRule is a test-only rule type for hello world advisor
	HelloWorldRule advisor.SQLReviewRuleType = "hello-world"
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, HelloWorldRule, &HelloWorldAdvisor{})
}

// HelloWorldAdvisor is a test advisor that always reports "hello world".
type HelloWorldAdvisor struct {
}

// Check always returns a warning at line 0 column 0 with message "hello world".
func (*HelloWorldAdvisor) Check(_ context.Context, _ advisor.Context) ([]*storepb.Advice, error) {
	return []*storepb.Advice{
		{
			Status:  storepb.Advice_WARNING,
			Code:    code.Ok.Int32(),
			Title:   "Hello World Test",
			Content: "hello world",
			StartPosition: &storepb.Position{
				Line:   0,
				Column: 0,
			},
		},
	}, nil
}
