package idp

import "testing"

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
