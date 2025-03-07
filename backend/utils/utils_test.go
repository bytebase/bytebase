package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetRenderedStatement(t *testing.T) {
	testCases := []struct {
		material map[string]string
		template string
		expected string
	}{
		{
			material: map[string]string{
				"PASSWORD": "123",
			},
			template: "select * from table where password = ${{ secrets.PASSWORD }}",
			expected: "select * from table where password = 123",
		},
		{
			material: map[string]string{
				"PASSWORD": "123",
				"USER":     `"abc"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("abc", 123)`,
		},
		// Replace recursively case.
		{
			material: map[string]string{
				"PASSWORD": "${{ secrets.USER }}",
				"USER":     `"abc"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("abc", ${{ secrets.USER }})`,
		},
		{
			material: map[string]string{
				"PASSWORD": "123",
				"USER":     `"${{ secrets.PASSWORD }}"`,
			},
			template: "INSERT INTO table (user, password) VALUES (${{ secrets.USER }}, ${{ secrets.PASSWORD }})",
			expected: `INSERT INTO table (user, password) VALUES ("123", 123)`,
		},
		{
			material: map[string]string{
				"USER": `"abc"`,
			},
			template: "select * from table where password = ${{ secrets.PASSWORD }}",
			expected: "select * from table where password = ${{ secrets.PASSWORD }}",
		},
	}

	for _, tc := range testCases {
		actual := RenderStatement(tc.template, tc.material)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestConvertBytesToUTF8String(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{
			input:    []byte{},
			expected: "",
		},
		{
			input:    []byte("hello"),
			expected: "hello",
		},
		{
			input:    []byte("你好"),
			expected: "你好",
		},
		{
			input:    []byte("Hello 世界 😊"),
			expected: "Hello 世界 😊",
		},
		{
			// string: SELECT "�ݱ�˼"
			input:    []byte{83, 69, 76, 69, 67, 84, 32, 34, 176, 221, 177, 180, 203, 188, 34},
			expected: "SELECT \"拜贝思\"",
		},
	}

	for _, test := range tests {
		actual, err := ConvertBytesToUTF8String(test.input)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, actual)
	}
}

func TestCheckDatabaseGroupMatch(t *testing.T) {
	tests := []struct {
		expression string
		database   *store.DatabaseMessage
		match      bool
	}{
		{
			expression: `resource.labels.unit == "gcp"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: true,
		},
		{
			expression: `resource.labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{
					Labels: map[string]string{
						"unit": "gcp",
					},
				},
			},
			match: false,
		},
		{
			expression: `has(resource.labels.unit) && resource.labels.unit == "aws"`,
			database: &store.DatabaseMessage{
				Metadata: &storepb.DatabaseMetadata{},
			},
			match: false,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		match, err := CheckDatabaseGroupMatch(ctx, test.expression, test.database)
		assert.NoError(t, err)
		assert.Equal(t, test.match, match)
	}
}
