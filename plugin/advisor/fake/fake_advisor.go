package fake

import (
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

type Advisor struct {
}

// A fake advisor to report 1 advice for each severity.
func (adv *Advisor) Check(ctx advisor.AdvisorContext, statement string) ([]advisor.Advice, error) {
	return []advisor.Advice{
		{
			Status:  advisor.Success,
			Title:   "INFO check",
			Content: "Fake check returnings INFO result",
		},
		{
			Status:  advisor.Warn,
			Title:   "WARN check",
			Content: "Fake check returnings WARN result",
		},
		{
			Status:  advisor.Error,
			Title:   "ERROR check",
			Content: "Fake check returnings ERROR result",
		},
	}, nil
}
