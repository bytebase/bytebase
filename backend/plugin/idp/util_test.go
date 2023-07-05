package idp

import "testing"

func TestGetValueWithKey(t *testing.T) {
	tests := []struct {
		data map[string]interface{}
		key  string
		want interface{}
	}{
		{
			data: map[string]interface{}{
				"user": "test",
			},
			key:  "user",
			want: "test",
		},
		{
			data: map[string]interface{}{
				"data": map[string]interface{}{
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
