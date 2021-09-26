package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerIssueSubscriberRoutes(g *echo.Group) {
	g.POST("/issue/:issueId/subscriber", func(c echo.Context) error {
		issueId, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
		}

		issueSubscriberCreate := &api.IssueSubscriberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueSubscriberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issueSubscriber request").SetInternal(err)
		}

		issueSubscriberCreate.IssueId = issueId

		issueSubscriber, err := s.IssueSubscriberService.CreateIssueSubscriber(context.Background(), issueSubscriberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Subscriber %d already exists in issue %d", issueSubscriberCreate.SubscriberId, issueSubscriberCreate.IssueId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d to issue %d", issueSubscriberCreate.SubscriberId, issueSubscriberCreate.IssueId)).SetInternal(err)
		}

		if err := s.ComposeIssueSubscriberRelationship(context.Background(), issueSubscriber); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber %d relationship for issue %d", issueSubscriberCreate.SubscriberId, issueSubscriberCreate.IssueId)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscriber); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue subscriber response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueId/subscriber", func(c echo.Context) error {
		issueId, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
		}

		issueSubscriberFind := &api.IssueSubscriberFind{
			IssueId: &issueId,
		}
		list, err := s.IssueSubscriberService.FindIssueSubscriberList(context.Background(), issueSubscriberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issueId)).SetInternal(err)
		}

		for _, issueSubscriber := range list {
			if err := s.ComposeIssueSubscriberRelationship(context.Background(), issueSubscriber); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber %d relationship for issue %d", issueSubscriber.SubscriberId, issueSubscriber.IssueId)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue subscriber list response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/issue/:issueId/subscriber/:subscriberId", func(c echo.Context) error {
		issueId, err := strconv.Atoi(c.Param("issueId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue ID is not a number: %s", c.Param("issueId"))).SetInternal(err)
		}

		subscriberId, err := strconv.Atoi(c.Param("subscriberId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Subscriber ID is not a number: %s", c.Param("subscriberId"))).SetInternal(err)
		}

		issueSubscriberDelete := &api.IssueSubscriberDelete{
			IssueId:      issueId,
			SubscriberId: subscriberId,
		}
		err = s.IssueSubscriberService.DeleteIssueSubscriber(context.Background(), issueSubscriberDelete)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Subscriber %d not found in issue %d", subscriberId, issueId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete subscriber %d from issue %d", subscriberId, issueId)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeIssueSubscriberRelationship(ctx context.Context, issueSubscriber *api.IssueSubscriber) error {
	var err error

	issueSubscriber.Subscriber, err = s.ComposePrincipalById(ctx, issueSubscriber.SubscriberId)
	if err != nil {
		return err
	}

	return nil
}
