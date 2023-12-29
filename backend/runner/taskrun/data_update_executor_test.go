package taskrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateToSelect(t *testing.T) {
	got, _ := updateToSelect("UPDATE hello SET column1 = value1, column2 = value2 WHERE id > 3;", "todozp", "113")
	assert.Equal(t, "CREATE TABLE `todozp`.`hello_113` LIKE hello; INSERT INTO `todozp`.`hello_113` SELECT * FROM hello WHERE id > 3;", got)
}
