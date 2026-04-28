package enterprise

import (
	"math"
	"testing"
)

func TestIsUnifiedInstanceLimit(t *testing.T) {
	tests := []struct {
		name           string
		instanceLimit  int
		activatedLimit int
		want           bool
	}{
		{name: "equal finite caps", instanceLimit: 10, activatedLimit: 10, want: true},
		{name: "activated cap larger than registration cap", instanceLimit: 10, activatedLimit: 20, want: true},
		{name: "split cap", instanceLimit: 50, activatedLimit: 20, want: false},
		{name: "unlimited both sides", instanceLimit: math.MaxInt, activatedLimit: math.MaxInt, want: true},
		{name: "unlimited registration finite activation", instanceLimit: math.MaxInt, activatedLimit: 20, want: false},
		{name: "finite registration unlimited activation", instanceLimit: 20, activatedLimit: math.MaxInt, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnifiedInstanceLimit(tt.instanceLimit, tt.activatedLimit); got != tt.want {
				t.Fatalf("isUnifiedInstanceLimit(%d, %d) = %v, want %v", tt.instanceLimit, tt.activatedLimit, got, tt.want)
			}
		})
	}
}
