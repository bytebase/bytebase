package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/stretchr/testify/assert"
)

func TestReassignEnvironmentID(t *testing.T) {
	tests := []struct {
		stageList []*api.Stage
		task      *api.Task
		want      int
	}{
		{
			stageList: []*api.Stage{
				{
					ID:            1,
					EnvironmentID: 1,
					TaskList: []*api.Task{
						{
							ID:      1,
							StageID: 1,
						},
						{
							ID:      2,
							StageID: 1,
						},
					},
				},
			},
			task: &api.Task{
				ID:      2,
				StageID: 1,
			},
			want: 1,
		},
		{
			stageList: []*api.Stage{
				{
					ID:            1,
					EnvironmentID: 1,
					TaskList: []*api.Task{
						{
							ID:      1,
							StageID: 1,
						},
						{
							ID:      2,
							StageID: 1,
						},
					},
				},
				{
					ID:            2,
					EnvironmentID: 2,
					TaskList: []*api.Task{
						{
							ID:      3,
							StageID: 2,
						},
						{
							ID:      4,
							StageID: 2,
						},
					},
				},
			},
			task: &api.Task{
				ID:      2,
				StageID: 1,
			},
			want: 2,
		},
		{
			stageList: []*api.Stage{
				{
					ID:            1,
					EnvironmentID: 1,
					TaskList: []*api.Task{
						{
							ID:      1,
							StageID: 1,
						},
						{
							ID:      2,
							StageID: 1,
						},
					},
				},
				{
					ID:            2,
					EnvironmentID: 2,
					TaskList: []*api.Task{
						{
							ID:      3,
							StageID: 2,
						},
						{
							ID:      4,
							StageID: 2,
						},
					},
				},
			},
			task: &api.Task{
				ID:      3,
				StageID: 2,
			},
			want: 2,
		},
		{
			stageList: []*api.Stage{
				{
					ID:            1,
					EnvironmentID: 1,
					TaskList: []*api.Task{
						{
							ID:      1,
							StageID: 1,
						},
						{
							ID:      2,
							StageID: 1,
						},
					},
				},
				{
					ID:            2,
					EnvironmentID: 2,
					TaskList: []*api.Task{
						{
							ID:      3,
							StageID: 2,
						},
						{
							ID:      4,
							StageID: 2,
						},
					},
				},
			},
			task: &api.Task{
				ID:      1,
				StageID: 1,
			},
			want: 1,
		},
	}

	for _, test := range tests {
		var environmentID int
		stageList := test.stageList
		task := test.task
		for i, stage := range stageList {
			if stage.ID == task.StageID {
				environmentID = stage.EnvironmentID
				if i != len(stageList)-1 && stage.TaskList[len(stage.TaskList)-1].ID == task.ID {
					environmentID = stageList[i+1].EnvironmentID
				}
				break
			}
		}
		assert.Equal(t, test.want, environmentID)
	}
}
