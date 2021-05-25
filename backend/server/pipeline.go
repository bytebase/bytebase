package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) ComposePipelineById(ctx context.Context, id int, includeList []string) (*api.Pipeline, error) {
	pipelineFind := &api.PipelineFind{
		ID: &id,
	}
	pipeline, err := s.PipelineService.FindPipeline(context.Background(), pipelineFind)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Pipeline ID not found: %d", id))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch pipeline ID: %v", id)).SetInternal(err)
	}

	if err := s.ComposePipelineRelationship(ctx, pipeline, includeList); err != nil {
		return nil, err
	}

	return pipeline, nil
}

func (s *Server) ComposePipelineRelationship(ctx context.Context, pipeline *api.Pipeline, includeList []string) error {
	var err error

	pipeline.Creator, err = s.ComposePrincipalById(context.Background(), pipeline.CreatorId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for pipeline: %v", pipeline.Name)).SetInternal(err)
	}

	pipeline.Updater, err = s.ComposePrincipalById(context.Background(), pipeline.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for pipeline: %v", pipeline.Name)).SetInternal(err)
	}

	pipeline.StageList, err = s.ComposeStageListByPipelineId(context.Background(), pipeline.ID, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch stage list for pipeline: %v", pipeline.Name)).SetInternal(err)
	}

	return nil
}

// Try to schedule the next task if needed
func (s *Server) ScheduleNextTaskIfNeeded(ctx context.Context, pipeline *api.Pipeline) error {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if !task.Status.IsEndStatus() {
				if task.When == api.TaskOnSuccess {
					_, err := s.TaskScheduler.Schedule(context.Background(), *task, api.SYSTEM_BOT_ID)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
	return nil
}
