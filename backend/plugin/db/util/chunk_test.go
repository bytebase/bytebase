package util

import (
	"fmt"
	"math/rand"
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

func TestRandomChunkedSQLScript(t *testing.T) {
	length := rand.Int()%10000 + 1
	var script []base.SingleSQL
	for i := 0; i < length; i++ {
		script = append(script, base.SingleSQL{
			Text: fmt.Sprintf("%d", rand.Int()),
		})
	}

	maxChunksCount := rand.Int()%200 + 1

	list, err := ChunkedSQLScript(script, maxChunksCount)
	require.NoErrorf(t, err, "length %d with maxChunksCount %d", length, maxChunksCount)
	require.LessOrEqualf(t, len(list), maxChunksCount, "length %d with maxChunksCount %d", length, maxChunksCount)
	id := 0
	for _, l := range list {
		for _, s := range l {
			require.Equalf(t, script[id].Text, s.Text, "length %d with maxChunksCount %d", length, maxChunksCount)
			id++
		}
	}
	require.Equalf(t, len(script), id, "length %d with maxChunksCount %d", length, maxChunksCount)
}
