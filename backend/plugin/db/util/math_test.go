package util

import "testing"

func TestCeilDivision(t *testing.T) {
	type args struct {
		dividend int
		divisor  int
	}
	tests := []struct {
		args    args
		want    int
		wantErr bool
	}{
		{
			args: args{
				dividend: 5,
				divisor:  2,
			},
			want:    3,
			wantErr: false,
		},
		{
			args: args{
				dividend: 2,
				divisor:  5,
			},
			want:    1,
			wantErr: false,
		},
		{
			args: args{
				dividend: 5,
				divisor:  5,
			},
			want:    1,
			wantErr: false,
		},
		{
			args: args{
				dividend: 0,
				divisor:  5,
			},
			want:    0,
			wantErr: false,
		},
		{
			args: args{
				dividend: 0,
				divisor:  0,
			},
			wantErr: true,
		},
		{
			args: args{
				dividend: 6,
				divisor:  5,
			},
			want:    2,
			wantErr: false,
		},
		{
			args: args{
				dividend: 4,
				divisor:  5,
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, test := range tests {
		got, err := CeilDivision(test.args.dividend, test.args.divisor)
		if (err != nil) != test.wantErr {
			t.Errorf("CeilDivision() error = %v, wantErr %v", err, test.wantErr)
			return
		}
		if got != test.want {
			t.Errorf("CeilDivision() = %v, want %v", got, test.want)
		}
	}
}
