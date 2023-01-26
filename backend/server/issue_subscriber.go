package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerIssueSubscriberRoutes(g *echo.Group) {
	g.POST("/issue/:issueID/subscriber", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueID})
		if err != nil {
			return err
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, "issue not found")
		}

		issueSubscriberCreate := &api.IssueSubscriberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueSubscriberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issueSubscriber request").SetInternal(err)
		}

		newSubscriber, err := s.store.GetUserByID(ctx, issueSubscriberCreate.SubscriberID)
		if err != nil {
			return err
		}
		if newSubscriber == nil {
			return echo.NewHTTPError(http.StatusNotFound, "subscriber not found")
		}
		newSubscribers := issue.Subscribers
		newSubscribers = append(newSubscribers, newSubscriber)

		if _, err := s.store.UpdateIssueV2(ctx, issueID, &store.UpdateIssueMessage{Subscribers: &newSubscribers}, api.SystemBotID); err != nil {
			return err
		}

		issueSubscriber := &api.IssueSubscriber{IssueID: issueID, SubscriberID: newSubscriber.ID}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscriber); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create issue subscriber response").SetInternal(err)
		}
		return nil
	})

	g.GET("/issue/:issueID/subscriber", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueID})
		if err != nil {
			return err
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, "issue not found")
		}

		var issueSubscribers []*api.IssueSubscriber
		for _, subscriber := range issue.Subscribers {
			issueSubscribers = append(issueSubscribers, &api.IssueSubscriber{
				IssueID:      issueID,
				SubscriberID: subscriber.ID,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscribers); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal issue subscriber list response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/issue/:issueID/subscriber/:subscriberID", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Issue ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueID})
		if err != nil {
			return err
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, "issue not found")
		}

		subscriberID, err := strconv.Atoi(c.Param("subscriberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Subscriber ID is not a number: %s", c.Param("subscriberID"))).SetInternal(err)
		}
		subscriber, err := s.store.GetUserByID(ctx, subscriberID)
		if err != nil {
			return err
		}
		if subscriber == nil {
			return echo.NewHTTPError(http.StatusNotFound, "subscriber not found")
		}

		var newSubscribers []*store.UserMessage
		for _, subscriber := range issue.Subscribers {
			if subscriber.ID == subscriberID {
				continue
			}
			newSubscribers = append(newSubscribers, subscriber)
		}
		if _, err := s.store.UpdateIssueV2(ctx, issueID, &store.UpdateIssueMessage{Subscribers: &newSubscribers}, api.SystemBotID); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}
