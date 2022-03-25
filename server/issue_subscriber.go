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
	g.POST("/issue/:issueID/subscriber", func(c echo.Context) error {
		ctx := context.Background()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueSubscriberCreate := &api.IssueSubscriberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueSubscriberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create issueSubscriber request").SetInternal(err)
		}

		issueSubscriberCreate.IssueID = issueID

		issueSubscriberRaw, err := s.IssueSubscriberService.CreateIssueSubscriber(ctx, issueSubscriberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Subscriber %d already exists in issue %d", issueSubscriberCreate.SubscriberID, issueSubscriberCreate.IssueID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d to issue %d", issueSubscriberCreate.SubscriberID, issueSubscriberCreate.IssueID)).SetInternal(err)
		}

		issueSubscriber, err := s.composeIssueSubscriberRelationship(ctx, issueSubscriberRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber %d relationship for issue %d", issueSubscriberCreate.SubscriberID, issueSubscriberCreate.IssueID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscriber); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue subscriber response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID/subscriber", func(c echo.Context) error {
		ctx := context.Background()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueSubscriberFind := &api.IssueSubscriberFind{
			IssueID: &issueID,
		}
		issueSubscriberRawList, err := s.IssueSubscriberService.FindIssueSubscriberList(ctx, issueSubscriberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issueID)).SetInternal(err)
		}

		var issueSubscriberList []*api.IssueSubscriber
		for _, issueSubscriberRaw := range issueSubscriberRawList {
			issueSubscriber, err := s.composeIssueSubscriberRelationship(ctx, issueSubscriberRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber %d relationship for issue %d", issueSubscriberRaw.SubscriberID, issueSubscriberRaw.IssueID)).SetInternal(err)
			}
			issueSubscriberList = append(issueSubscriberList, issueSubscriber)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscriberList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue subscriber list response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/issue/:issueID/subscriber/:subscriberID", func(c echo.Context) error {
		ctx := context.Background()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		subscriberID, err := strconv.Atoi(c.Param("subscriberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Subscriber ID is not a number: %s", c.Param("subscriberID"))).SetInternal(err)
		}

		issueSubscriberDelete := &api.IssueSubscriberDelete{
			IssueID:      issueID,
			SubscriberID: subscriberID,
		}
		if err := s.IssueSubscriberService.DeleteIssueSubscriber(ctx, issueSubscriberDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete subscriber %d from issue %d", subscriberID, issueID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeIssueSubscriberRelationship(ctx context.Context, raw *api.IssueSubscriberRaw) (*api.IssueSubscriber, error) {
	issueSubscriber := raw.ToIssueSubscriber()

	subscriber, err := s.store.GetPrincipalByID(ctx, issueSubscriber.SubscriberID)
	if err != nil {
		return nil, err
	}
	issueSubscriber.Subscriber = subscriber

	return issueSubscriber, nil
}
