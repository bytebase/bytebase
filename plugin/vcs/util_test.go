package vcs

import (
	"testing"
)

func TestBranch(t *testing.T) {
	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{
			ref:     "refs/heads/master",
			want:    "master",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
	}

	for _, test := range tests {
		result, err := Branch(test.ref)
		if err != nil && !test.wantErr {
			t.Errorf("Branch %q: got error %v, want OK.", test.ref, test.wantErr)
		}

		if err == nil && test.wantErr {
			t.Errorf("Branch %q: got OK, want error", test.ref)
		}

		if result != test.want {
			t.Errorf("Branch %q: got result %v, want %v.", test.ref, result, test.want)
		}
	}
}
