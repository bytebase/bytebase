package taskrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateToSelect(t *testing.T) {
	got := updateToSelect("UPDATE hello SET column1 = value1, column2 = value2 WHERE id > 3;", "todozp", "113")
	assert.Equal(t, "CREATE TABLE todozp.hello-113 AS SELECT * FROM hello WHERE id > 3;", got)
}
