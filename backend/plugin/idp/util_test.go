package idp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetValueWithKey(t *testing.T) {
	tests := []struct {
		data map[string]any
		key  string
		want any
	}{
		{
			data: map[string]any{
				"user": "test",
			},
			key:  "user",
			want: "test",
		},
		{
			data: map[string]any{
				"data": map[string]any{
					"user": "test",
				},
			},
			key:  "data.user",
			want: "test",
		},
	}

	for _, test := range tests {
		got := GetValueWithKey(test.data, test.key)
		if test.want != got {
			t.Errorf("GetValueWithKey got %v but want %v", got, test.want)
		}
	}
}

func TestGetValueWithKeyFromJSONString(t *testing.T) {
	tests := []struct {
		data string
		key  string
		want any
	}{
		{
			data: `{"user": "test"}`,
			key:  "user",
			want: "test",
		},
		{
			data: `{"user":"test","data":{"user":"data-user"}}`,
			key:  "data.user",
			want: "data-user",
		},
	}

	for _, test := range tests {
		var data map[string]any
		err := json.Unmarshal([]byte(test.data), &data)
		require.NoError(t, err)
		got := GetValueWithKey(data, test.key)
		if test.want != got {
			t.Errorf("GetValueWithKey got %v but want %v", got, test.want)
		}
	}
}
