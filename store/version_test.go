package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_versionFromInt(t *testing.T) {
	t.Run("should parse major and minor correctly", func(t *testing.T) {
		tests := []struct {
			num, major, minor int
		}{
			{10031, 1, 31},
			{131, 0, 131},
			{20009, 2, 9},
		}
		for _, tt := range tests {
			v := versionFromInt(tt.num)
			assert.Equal(t, v.major, tt.major)
			assert.Equal(t, v.minor, tt.minor)
		}
	})
}

func Test_version_biggerThan(t *testing.T) {
	type fields struct {
		major int
		minor int
	}
	type args struct {
		other version
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should return true if major is bigger",
			fields: fields{
				major: 1,
				minor: 0,
			},
			args: args{
				other: version{
					major: 0,
					minor: 0,
				},
			},
			want: true,
		},
		{
			name: "should return true if major and minor are bigger",
			fields: fields{
				major: 1,
				minor: 1,
			},
			args: args{
				other: version{
					major: 1,
					minor: 0,
				},
			},
			want: true,
		},
		{
			name: "should return false if major is smaller",
			fields: fields{
				major: 0,
				minor: 0,
			},
			args: args{
				other: version{
					major: 1,
					minor: 0,
				},
			},
			want: false,
		},
		{
			name: "should return false if major and minor are smaller",
			fields: fields{
				major: 0,
				minor: 0,
			},
			args: args{
				other: version{
					major: 1,
					minor: 1,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := version{
				major: tt.fields.major,
				minor: tt.fields.minor,
			}
			got := v.biggerThan(tt.args.other)
			assert.Equal(t, got, tt.want)
		})
	}
}
