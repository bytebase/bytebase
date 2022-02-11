package fake

import (
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
)

var (
	_ advisor.Advisor = (*Advisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.Fake, &Advisor{})
	advisor.Register(db.Postgres, advisor.Fake, &Advisor{})
	advisor.Register(db.TiDB, advisor.Fake, &Advisor{})
}

// Advisor is the fake sql advisor.
type Advisor struct {
}

// Check is a fake advisor check reporting 1 advice for each severity.
func (adv *Advisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	return []advisor.Advice{
		{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "INFO check",
			Content: statement,
		},
		{
			Status:  advisor.Warn,
			Code:    common.Internal,
			Title:   "WARN check",
			Content: statement,
		},
		{
			Status:  advisor.Error,
			Code:    common.Internal,
			Title:   "ERROR check",
			Content: statement,
		},
	}, nil
}
