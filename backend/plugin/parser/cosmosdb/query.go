package cosmosdb

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_COSMOSDB, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	_, err := ParseCosmosDBQuery(statement)
	if err != nil {
		return false, false, err
	}

	return true, true, nil
}
