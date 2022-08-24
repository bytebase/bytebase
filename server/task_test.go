package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/assert"
)

func TestIsTaskAllDone(t *testing.T) {
	tests := []struct {
		pipeline *api.Pipeline
		want     bool
	}{
		{
			pipeline: &api.Pipeline{
				StageList: []*api.Stage{
					{
						TaskList: []*api.Task{
							{
								Status: api.TaskDone,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			pipeline: &api.Pipeline{
				StageList: []*api.Stage{
					{
						TaskList: []*api.Task{
							{
								Status: api.TaskDone,
							},
						},
					},
					{
						TaskList: []*api.Task{
							{
								Status: api.TaskDone,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			pipeline: &api.Pipeline{
				StageList: []*api.Stage{
					{
						TaskList: []*api.Task{
							{
								Status: api.TaskDone,
							},
						},
					},
					{
						TaskList: []*api.Task{
							{
								Status: api.TaskDone,
							},
							{
								Status: api.TaskFailed,
							},
							{
								Status: api.TaskPending,
							},
							{
								Status: api.TaskRunning,
							},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		res := isTaskAllDone(test.pipeline)
		assert.Equal(t, test.want, res)
	}
}
