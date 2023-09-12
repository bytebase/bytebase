package server

import (
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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *Server) registerIssueRoutes(g *echo.Group) {
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
			if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
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

		if err := utils.ChangeIssueStatus(ctx, s.store, s.activityManager, issue, issueStatusPatch.Status, issueStatusPatch.UpdaterID, issueStatusPatch.Comment); err != nil {
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
