package sqlserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	metricAPI "github.com/bytebase/bytebase/metric"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/metric"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct{}

// GetDatabase is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetDatabase() *catalog.Database {
	return &catalog.Database{}
}

// GetDatabase is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetFinder() *catalog.Finder {
	return catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: false})
}

type sqlCheckRequestBody struct {
	Statement    string                      `json:"statement"`
	DatabaseType string                      `json:"databaseType"`
	TemplateID   advisor.SQLReviewTemplateID `json:"templateId"`
	Override     string                      `json:"override"`
}

func (s *Server) registerAdvisorRoutes(g *echo.Group) {
	g.POST("/advise", s.sqlCheckController)
}

// sqlCheckController godoc
// @Summary  Check the SQL statement.
// @Description  Parse and check the SQL statement according to the SQL review rules.
// @Accept  application/json
// @Tags  SQL review
// @Produce  json
// @Param  statement     body  string  true   "The SQL statement."
// @Param  databaseType  body  string  true   "The database type."  Enums(MYSQL, POSTGRES, TIDB)
// @Param  templateId    body  string  false  "The SQL check template id. Required if the config is not specified." Enums(bb.sql-review.prod, bb.sql-review.dev)
// @Param  override      body  string  false  "The SQL check config override string in YAML format. Check https://github.com/bytebase/bytebase/tree/main/plugin/advisor/config/sql-review.override.yaml for example. Required if the template is not specified."
// @Success  200  {array}   advisor.Advice
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /advise  [post].
func (s *Server) sqlCheckController(c echo.Context) error {
	request := &sqlCheckRequestBody{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	if err := json.Unmarshal(body, request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot format request body").SetInternal(err)
	}

	if request.Statement == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required SQL statement")
	}

	if request.Override == "" && request.TemplateID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required template or override")
	}

	advisorDBType, err := advisorDB.ConvertToAdvisorDBType(request.DatabaseType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", request.DatabaseType))
	}

	ruleOverride := &advisor.SQLReviewConfigOverride{}
	if request.Override != "" {
		if err := yaml.Unmarshal([]byte(request.Override), ruleOverride); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid config: %v", request.Override)).SetInternal(err)
		}
		if request.TemplateID != "" && ruleOverride.Template != request.TemplateID {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The config override should extend from the same template. Found %s in override but also get %s template in request.", ruleOverride.Template, request.TemplateID))
		}
	} else {
		ruleOverride.Template = request.TemplateID
	}

	ruleList, err := advisor.MergeSQLReviewRules(ruleOverride)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot merge the config for template: %s", ruleOverride.Template)).SetInternal(err)
	}

	adviceList, err := sqlCheck(
		advisorDBType,
		"utf8mb4",
		"utf8mb4_general_ci",
		request.Statement,
		ruleList,
		&catalogService{},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to run sql check").SetInternal(err)
	}

	s.metricReporter.Report(&metric.Metric{
		Name:  metricAPI.SQLAdviseAPIMetricName,
		Value: 1,
		Labels: map[string]string{
			"database_type": string(advisorDBType),
			"platform":      c.Request().Header.Get("X-Platform"),
			"repository":    c.Request().Header.Get("X-Repository"),
			"actor":         c.Request().Header.Get("X-Actor"),
			"source":        c.Request().Header.Get("X-Source"),
			"version":       c.Request().Header.Get("X-Version"),
		},
	})

	return c.JSON(http.StatusOK, adviceList)
}

func sqlCheck(
	dbType advisorDB.Type,
	dbCharacterSet string,
	dbCollation string,
	statement string,
	ruleList []*advisor.SQLReviewRule,
	catalog catalog.Catalog,
) ([]advisor.Advice, error) {
	var adviceList []advisor.Advice

	res, err := advisor.SQLReviewCheck(statement, ruleList, advisor.SQLReviewCheckContext{
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
