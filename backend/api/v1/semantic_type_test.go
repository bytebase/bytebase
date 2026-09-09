package v1

import "testing"

func TestIsBuiltinSemanticTypeID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{id: defaultSemanticTypeID, want: true},
		{id: defaultPartialSemanticTypeID, want: true},
		{id: "email", want: false},
		{id: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			if got := isBuiltinSemanticTypeID(test.id); got != test.want {
				t.Fatalf("isBuiltinSemanticTypeID(%q) = %v, want %v", test.id, got, test.want)
			}
		})
	}
}
