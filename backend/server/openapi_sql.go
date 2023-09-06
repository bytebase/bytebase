package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"
)

type schemaDiffRequestBody struct {
	EngineType   parser.EngineType `json:"engineType"`
	SourceSchema string            `json:"sourceSchema"`
	TargetSchema string            `json:"targetSchema"`
}

// schemaDiff godoc
// @Summary  Get the diff statement between source and target schema.
// @Description  Parse and diff the schema statements.
// @Accept  */*
// @Tags  SQL schema diff
// @Produce  json
// @Param  engineType       body  string  true   "The database engine type."
// @Param  sourceSchema     body  string  true   "The source schema statement."
// @Param  targetSchema     body  string  false  "The target schema statement."
// @Success  200  {string}  the target diff string of schemas
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/schema/diff  [post].
func schemaDiff(c echo.Context) error {
	request := &schemaDiffRequestBody{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	if err := json.Unmarshal(body, request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot format request body").SetInternal(err)
	}

	var engine parser.EngineType
	switch request.EngineType {
	case parser.EngineType(db.Postgres):
		engine = parser.Postgres
	case parser.EngineType(db.MySQL), parser.EngineType(db.MariaDB), parser.EngineType(db.OceanBase):
		engine = parser.MySQL
	case parser.EngineType(db.TiDB):
		engine = parser.TiDB
	default:
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid database engine %s", request.EngineType))
	}

	diff, err := differ.SchemaDiff(engine, request.SourceSchema, request.TargetSchema, false /* ignoreCaseSensitive */)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compute diff between source and target schemas").SetInternal(err)
	}

	return c.JSON(http.StatusOK, diff)
}
