package sqlreviewserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct{}

// GetDatabase is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetDatabase(_ context.Context) (*catalog.Database, error) {
	return nil, nil
}

func (s *Server) registerAdvisorRoutes(g *echo.Group) {
	g.GET("/sql/advise", s.sqlCheckController)
}

// sqlCheckController godoc
// @Summary  Check the SQL statement.
// @Description  Parse and check the SQL statement according to the schema review rules.
// @Accept  */*
// @Tags  Schema Review
// @Produce  json
// @Param  statement     query  string  true   "The SQL statement."
// @Param  databaseType  query  string  true   "The database type."  Enums(MySQL, PostgreSQL, TiDB)
// @Param  template      query  string  false  "The SQL check template id. Required if the config is not specified." Enums(bb.sql-review.mysql.prod, bb.sql-review.mysql.dev)
// @Param  config        query  string  false  "The SQL check config string in YAML format. Required if the template is not specified."
// @Success  200  {array}   advisor.Advice
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/advise  [get].
func (s *Server) sqlCheckController(c echo.Context) error {
	statement := c.QueryParams().Get("statement")
	if statement == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required SQL statement")
	}

	databaseType := c.QueryParams().Get("databaseType")
	if databaseType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required database type")
	}
	dbType := db.Type(strings.ToUpper(databaseType))
	advisorDBType, err := api.ConvertToAdvisorDBType(dbType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", dbType))
	}

	template := c.QueryParams().Get("template")
	configOverrideYAMLStr := c.QueryParams().Get("config")
	if template == "" && configOverrideYAMLStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required template or config")
	}

	ruleOverride := &advisor.SQLReviewConfigOverride{}
	if configOverrideYAMLStr != "" {
		if err := yaml.Unmarshal([]byte(configOverrideYAMLStr), ruleOverride); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid config %v", configOverrideYAMLStr)).SetInternal(err)
		}
		if template != "" && string(ruleOverride.Template) != template {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The config should extend from the same template. Found %s in config but also get %s template in request.", ruleOverride.Template, template))
		}
	} else {
		ruleOverride.Template = advisor.SQLReviewTemplateID(template)
	}

	ruleList, err := advisor.MergeSQLReviewRules(ruleOverride)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot merge the config for template %s", ruleOverride.Template)).SetInternal(err)
	}

	ctx := c.Request().Context()
	adviceList, err := sqlCheck(
		ctx,
		advisorDBType,
		"utf8mb4",
		"utf8mb4_general_ci",
		statement,
		ruleList,
		&catalogService{},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to run sql check").SetInternal(err)
	}

	return c.JSON(http.StatusOK, adviceList)
}

func sqlCheck(
	ctx context.Context,
	dbType advisor.DBType,
	dbCharacterSet string,
	dbCollation string,
	statement string,
	ruleList []*advisor.SQLReviewRule,
	catalog catalog.Catalog,
) ([]advisor.Advice, error) {
	var adviceList []advisor.Advice

	res, err := advisor.SchemaReviewCheck(statement, ruleList, advisor.SQLReviewCheckContext{
		Charset:   dbCharacterSet,
		Collation: dbCollation,
		DbType:    dbType,
		Catalog:   catalog,
	})
	if err != nil {
		return nil, err
	}

	adviceLevel := advisor.Success
	for _, advice := range res {
		switch advice.Status {
		case advisor.Warn:
			if adviceLevel != advisor.Error {
				adviceLevel = advisor.Warn
			}
		case advisor.Error:
			adviceLevel = advisor.Error
		case advisor.Success:
			continue
		}

		adviceList = append(adviceList, advice)
	}

	if len(adviceList) == 0 {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return adviceList, nil
}
