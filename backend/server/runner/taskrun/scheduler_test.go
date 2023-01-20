package taskrun

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/api"
)

func TestAreAllTasksDone(t *testing.T) {
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
		res := areAllTasksDone(test.pipeline)
		assert.Equal(t, test.want, res)
	}
}
