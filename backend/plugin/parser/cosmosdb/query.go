package cosmosdb

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
