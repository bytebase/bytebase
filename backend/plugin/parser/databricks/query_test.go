package databricks

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateQuerySeam(t *testing.T) {
	require.False(t, base.HasQueryValidator(storepb.Engine_DATABRICKS))
}
