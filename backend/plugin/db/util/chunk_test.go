package util

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestChunkedSQLScript(t *testing.T) {
	type args struct {
		script         []base.SingleSQL
		maxChunksCount int
	}
	tests := []struct {
		args    args
		want    [][]base.SingleSQL
		wantErr bool
	}{
		{
			args: args{
				script: []base.SingleSQL{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
					{
						Text: "3",
					},
					{
						Text: "4",
					},
					{
						Text: "5",
					},
				},
				maxChunksCount: 2,
			},
			want: [][]base.SingleSQL{
				{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
					{
						Text: "3",
					},
				},
				{
					{
						Text: "4",
					},
					{
						Text: "5",
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				script: []base.SingleSQL{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
					{
						Text: "3",
					},
					{
						Text: "4",
					},
					{
						Text: "5",
					},
				},
				maxChunksCount: 3,
			},
			want: [][]base.SingleSQL{
				{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
				},
				{
					{
						Text: "3",
					},
					{
						Text: "4",
					},
				},
				{
					{
						Text: "5",
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				script: []base.SingleSQL{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
					{
						Text: "3",
					},
					{
						Text: "4",
					},
					{
						Text: "5",
					},
				},
				maxChunksCount: 5,
			},
			want: [][]base.SingleSQL{

				{
					{
						Text: "1",
					},
				},
				{
					{
						Text: "2",
					},
				},
				{
					{
						Text: "3",
					},
				},
				{
					{
						Text: "4",
					},
				},
				{
					{
						Text: "5",
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				script: []base.SingleSQL{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
					{
						Text: "3",
					},
					{
						Text: "4",
					},
					{
						Text: "5",
					},
				},
				maxChunksCount: 4,
			},
			want: [][]base.SingleSQL{

				{
					{
						Text: "1",
					},
					{
						Text: "2",
					},
				},
				{
					{
						Text: "3",
					},
				},
				{
					{
						Text: "4",
					},
				},
				{
					{
						Text: "5",
					},
				},
			},
			wantErr: false,
		},
	}

	for i, tt := range tests {
		got, err := ChunkedSQLScript(tt.args.script, tt.args.maxChunksCount)
		if (err != nil) != tt.wantErr {
			t.Errorf("ChunkedSQLScript() error = %v, wantErr %v", err, tt.wantErr)
			continue
		}
		require.Equal(t, tt.want, got, i)
	}
}
