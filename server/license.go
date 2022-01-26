package server

import (
	"net/http"

	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerLicenseRoutes(g *echo.Group) {
	g.GET("/license", func(c echo.Context) error {
		license, err := s.LicenseService.ParseLicense()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot find valid license").SetInternal(err)
		}

		s.license = license

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, license); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal license response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/license", func(c echo.Context) error {
		patch := &enterpriseAPI.LicensePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create license request").SetInternal(err)
		}

		if err := s.LicenseService.StoreLicense(patch.Token); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create license").SetInternal(err)
		}

		license, err := s.LicenseService.ParseLicense()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot find valid license").SetInternal(err)
		}

		s.license = license

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, license); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal license response").SetInternal(err)
		}
		return nil
	})
}
