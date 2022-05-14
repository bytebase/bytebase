package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

func (s *Server) registerIssueSubscriberRoutes(g *echo.Group) {
	g.POST("/issue/:issueID/subscriber", func(c echo.Context) error {
		ctx := c.Request().Context()
		issueID, err := strconv.Atoi(c.Param("issueID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("issueID"))).SetInternal(err)
		}

		issueSubscriberCreate := &api.IssueSubscriberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, issueSubscriberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create issueSubscriber request").SetInternal(err)
		}

		issueSubscriberCreate.IssueID = issueID

		issueSubscriber, err := s.store.CreateIssueSubscriber(ctx, issueSubscriberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Subscriber %d already exists in issue %d", issueSubscriberCreate.SubscriberID, issueSubscriberCreate.IssueID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to add subscriber %d to issue %d", issueSubscriberCreate.SubscriberID, issueSubscriberCreate.IssueID)).SetInternal(err)
		}

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

		issueSubscriberFind := &api.IssueSubscriberFind{
			IssueID: &issueID,
		}
		issueSubscriberList, err := s.store.FindIssueSubscriber(ctx, issueSubscriberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch subscriber list for issue %d", issueID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, issueSubscriberList); err != nil {
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

		subscriberID, err := strconv.Atoi(c.Param("subscriberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Subscriber ID is not a number: %s", c.Param("subscriberID"))).SetInternal(err)
		}

		issueSubscriberDelete := &api.IssueSubscriberDelete{
			IssueID:      issueID,
			SubscriberID: subscriberID,
		}
		if err := s.store.DeleteIssueSubscriber(ctx, issueSubscriberDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete subscriber %d from issue %d", subscriberID, issueID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}
