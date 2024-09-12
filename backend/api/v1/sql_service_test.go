package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOffsetAndOriginTable(t *testing.T) {
	tests := []struct {
		input     string
		offset    int
		tableName string
	}{
		{
			input:     "_20240614164851_0_TEST_table1",
			offset:    0,
			tableName: "TEST_table1",
		},
		{
			input:     "_20240614164851_11_TEST",
			offset:    11,
			tableName: "TEST",
		},
		{
			input:     "_20240614164851_8_TEST_input_table1",
			offset:    8,
			tableName: "TEST_input_table1",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		offset, tabaleName, err := getOffsetAndOriginTable(test.input)
		a.NoError(err)
		a.Equal(test.offset, offset)
		a.Equal(test.tableName, tabaleName)
	}
}
