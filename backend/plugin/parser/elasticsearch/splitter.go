package elasticsearch

import (
	es "github.com/bytebase/omni/elasticsearch"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ELASTICSEARCH, SplitMultiSQL)
}

// SplitMultiSQL splits the input into individual ElasticSearch REST API requests.
func SplitMultiSQL(statement string) ([]base.Statement, error) {
	omniStmts, err := es.SplitMultiSQL(statement)
	if err != nil {
		return nil, err
	}
	if omniStmts == nil {
		return nil, nil
	}

	var statements []base.Statement
	for _, s := range omniStmts {
		statements = append(statements, base.Statement{
			Text:  s.Text,
			Empty: s.Empty,
			Start: &storepb.Position{
				Line:   int32(s.Start.Line),
				Column: int32(s.Start.Column),
			},
			End: &storepb.Position{
				Line:   int32(s.End.Line),
				Column: int32(s.End.Column),
			},
			Range: &storepb.Range{
				Start: int32(s.Range.Start),
				End:   int32(s.Range.End),
			},
		})
	}
	return statements, nil
}
