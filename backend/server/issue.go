package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
	g.POST("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueCreate := &api.IssueCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issue request").SetInternal(err)
		}

		if issueCreate.ProjectID == api.DefaultProjectUID {
			return echo.NewHTTPError(http.StatusBadRequest, "Cannot create a new issue in the default project")
		}
		if issueCreate.Type != api.IssueGrantRequest {
			return errors.Errorf("unsupported issue type %q", issueCreate.Type)
		}
		issue, err := s.createGrantRequestIssue(ctx, issueCreate, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			return err
		}

		s.MetricReporter.Report(ctx, &metric.Metric{
			Name:  metricAPI.IssueCreateMetricName,
			Value: 1,
			Labels: map[string]any{
				"type": issue.Type,
			},
		})

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueFind := &store.FindIssueMessage{}

		pageToken := c.QueryParams().Get("token")
		// We use descending order by default for issues.
		sinceID, err := unmarshalPageToken(pageToken, api.DESC)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed page token").SetInternal(err)
		}
		issueFind.SinceID = &sinceID

		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			issueFind.ProjectUID = &projectID
		}
		if issueStatusListStr := c.QueryParam("status"); issueStatusListStr != "" {
			statusList := []api.IssueStatus{}
			for _, status := range strings.Split(issueStatusListStr, ",") {
				statusList = append(statusList, api.IssueStatus(status))
			}
			issueFind.StatusList = statusList
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			issueFind.Limit = &limit
		} else {
			limit := api.DefaultPageSize
			issueFind.Limit = &limit
		}

		if userIDStr := c.QueryParam("user"); userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("user query parameter is not a number: %s", userIDStr)).SetInternal(err)
			}
			issueFind.PrincipalID = &userID
		}
		if creatorIDStr := c.QueryParam("creator"); creatorIDStr != "" {
			creatorID, err := strconv.Atoi(creatorIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("creator query parameter is not a number: %s", creatorIDStr)).SetInternal(err)
			}
			issueFind.CreatorID = &creatorID
		}
		if assigneeIDStr := c.QueryParam("assignee"); assigneeIDStr != "" {
			assigneeID, err := strconv.Atoi(assigneeIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("assignee query parameter is not a number: %s", assigneeIDStr)).SetInternal(err)
			}
			issueFind.AssigneeID = &assigneeID
		}
		if subscriberIDStr := c.QueryParam("subscriber"); subscriberIDStr != "" {
			subscriberID, err := strconv.Atoi(subscriberIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("subscriber query parameter is not a number: %s", subscriberIDStr)).SetInternal(err)
			}
			issueFind.SubscriberID = &subscriberID
		}

		issueList, err := s.store.FindIssueStripped(ctx, issueFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch issue list").SetInternal(err)
		}

		for _, issue := range issueList {
			if issue.Pipeline == nil {
				continue
			}
			for _, stage := range issue.Pipeline.StageList {
				for _, task := range stage.TaskList {
					switch task.LatestTaskRunStatus {
					case api.TaskRunNotStarted:
						task.Status = api.TaskPendingApproval
					case api.TaskRunPending:
						task.Status = api.TaskPending
					case api.TaskRunRunning:
						task.Status = api.TaskRunning
					case api.TaskRunDone:
						task.Status = api.TaskDone
					case api.TaskRunFailed:
						task.Status = api.TaskFailed
					case api.TaskRunCanceled:
						task.Status = api.TaskCanceled
					}
				}
			}
		}

		issueResponse := &api.IssueResponse{}
		issueResponse.Issues = issueList

		nextSinceID := sinceID
		if len(issueList) > 0 {
			// Decrement the ID as we use decreasing order by default.
			nextSinceID = issueList[len(issueList)-1].ID - 1
		}
		if issueResponse.NextToken, err = marshalPageToken(nextSinceID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal page token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueResponse); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		issuePatch := &api.IssuePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issuePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update issue request").SetInternal(err)
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when updating issue: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Unable to find issue ID to update: %d", id))
		}
		updateIssueMessage := &store.UpdateIssueMessage{
			Title:       issuePatch.Name,
			Description: issuePatch.Description,
			Payload:     issuePatch.Payload,
		}

		if issuePatch.AssigneeID != nil {
			assignee, err := s.store.GetUserByID(ctx, *issuePatch.AssigneeID)
			if err != nil {
				return err
			}
			updateIssueMessage.Assignee = assignee
		}

		updatedIssue, err := s.store.UpdateIssueV2(ctx, id, updateIssueMessage, updaterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update issue with ID %d", id)).SetInternal(err)
		}

		payloadList := [][]byte{}
		if issuePatch.Name != nil && *issuePatch.Name != issue.Title {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldName,
				OldValue:  issue.Title,
				NewValue:  *issuePatch.Name,
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue name: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.Description != nil && *issuePatch.Description != issue.Description {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldDescription,
				OldValue:  issue.Description,
				NewValue:  *issuePatch.Description,
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue description: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}
		if issuePatch.AssigneeID != nil && *issuePatch.AssigneeID != issue.Assignee.ID {
			payload, err := json.Marshal(api.ActivityIssueFieldUpdatePayload{
				FieldID:   api.IssueFieldAssignee,
				OldValue:  strconv.Itoa(issue.Assignee.ID),
				NewValue:  strconv.Itoa(*issuePatch.AssigneeID),
				IssueName: issue.Title,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing issue assignee: %v", updatedIssue.Title)).SetInternal(err)
			}
			payloadList = append(payloadList, payload)
		}

		for _, payload := range payloadList {
			activityCreate := &store.ActivityMessage{
				CreatorUID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerUID: issue.UID,
				Type:         api.ActivityIssueFieldUpdate,
				Level:        api.ActivityInfo,
				Payload:      string(payload),
			}
			if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: updatedIssue,
			}); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating issue: %v", updatedIssue.Title)).SetInternal(err)
			}
		}

		composedIssue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update issue response: %v", updatedIssue.Title)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/issue/:issueID/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueStatusPatch := &api.IssueStatusPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update issue status request").SetInternal(err)
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID: %v", id)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Issue ID not found: %d", id))
		}

		if err := utils.ChangeIssueStatus(ctx, s.store, s.ActivityManager, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound).SetInternal(err)
			} else if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		updatedComposedIssue, err := s.store.GetIssueByID(ctx, id)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedComposedIssue); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal issue ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) createGrantRequestIssue(ctx context.Context, issueCreate *api.IssueCreate, creatorID int) (*api.Issue, error) {
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &issueCreate.ProjectID})
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project %d not found", issueCreate.ProjectID))
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, err
	}
	var assignee *store.UserMessage
	for _, binding := range policy.Bindings {
		if binding.Role != api.Owner {
			continue
		}
		if binding.Condition == nil || binding.Condition.Expression != "" {
			continue
		}
		if len(binding.Members) == 0 {
			continue
		}
		assignee = binding.Members[0]
		break
	}
	if assignee == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project owner %d not found", issueCreate.ProjectID))
	}

	var issuePayload storepb.IssuePayload
	if err := protojson.Unmarshal([]byte(issueCreate.Payload), &issuePayload); err != nil {
		return nil, err
	}
	issueCreatePayload := &storepb.IssuePayload{
		GrantRequest: issuePayload.GrantRequest,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
		},
	}
	issueCreatePayloadBytes, err := protojson.Marshal(issueCreatePayload)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue payload").SetInternal(err)
	}
	issueCreateMessage := &store.IssueMessage{
		Project:     project,
		Title:       issueCreate.Name,
		Type:        issueCreate.Type,
		Description: issueCreate.Description,
		Assignee:    assignee,
		Payload:     string(issueCreatePayloadBytes),
	}
	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, creatorID)
	if err != nil {
		return nil, err
	}
	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Title,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   creatorID,
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueCreate,
		Level:        api.ActivityInfo,
		Payload:      string(bytes),
	}
	if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: issue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}

	// Composed the issue.
	composedIssue := &api.Issue{
		ID:          issue.UID,
		CreatorID:   issue.Creator.ID,
		CreatedTs:   issue.CreatedTime.Unix(),
		UpdaterID:   issue.Updater.ID,
		UpdatedTs:   issue.UpdatedTime.Unix(),
		ProjectID:   issue.Project.UID,
		PipelineID:  issue.PipelineUID,
		Name:        issue.Title,
		Status:      issue.Status,
		Type:        issue.Type,
		Description: issue.Description,
		AssigneeID:  issue.Assignee.ID,
		Payload:     issue.Payload,
	}
	composedCreator, err := s.store.GetPrincipalByID(ctx, issue.Creator.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Creator = composedCreator
	composedUpdater, err := s.store.GetPrincipalByID(ctx, issue.Updater.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Updater = composedUpdater
	composedAssignee, err := s.store.GetPrincipalByID(ctx, issue.Assignee.ID)
	if err != nil {
		return nil, err
	}
	composedIssue.Assignee = composedAssignee
	return composedIssue, nil
}

func marshalPageToken(id int) (string, error) {
	b, err := json.Marshal(id)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal page token")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// unmarshalPageToken unmarshals the page token, and returns the last issue ID. If the page token nil empty, it returns MaxInt32.
func unmarshalPageToken(pageToken string, sortOrder api.SortOrder) (int, error) {
	if pageToken == "" {
		if sortOrder == api.ASC {
			return 0, nil
		}
		return math.MaxInt32, nil
	}

	bs, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return 0, errors.Wrap(err, "failed to decode page token")
	}

	var id int
	if err := json.Unmarshal(bs, &id); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal page token")
	}

	return id, nil
}
