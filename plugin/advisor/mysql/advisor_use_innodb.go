package mysql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
)

const (
	innoDB              string = "innodb"
	defaultStorageEngin string = "default_storage_engine"
)

var (
	_ advisor.Advisor = (*UseInnoDBAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLUseInnoDB, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (adv *UseInnoDBAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &useInnoDBChecker{
		level: level,
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type useInnoDBChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
}

// Enter implements the ast.Visitor interface
func (v *useInnoDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := common.Ok
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, option := range node.Options {
			if option.Tp == ast.TableOptionEngine && strings.ToLower(option.StrValue) != innoDB {
				code = common.NotInnoDBEngine
				break
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			// TABLE OPTION
			if spec.Tp == ast.AlterTableOption {
				for _, option := range spec.Options {
					if option.Tp == ast.TableOptionEngine && strings.ToLower(option.StrValue) != innoDB {
						code = common.NotInnoDBEngine
						break
					}
				}
			}
		}
	// SET
	case *ast.SetStmt:
		for _, variable := range node.Variables {
			if strings.ToLower(variable.Name) == defaultStorageEngin {
				// Return lowercase
				text, err := restoreNode(variable.Value, format.RestoreNameLowercase)
				if err != nil {
					v.adviceList = append(v.adviceList, advisor.Advice{
						Status:  v.level,
						Code:    common.Internal,
						Title:   "Internal error for use InnoDB rule",
						Content: fmt.Sprintf("%q meet internal error %q", in.Text(), err.Error()),
					})
					continue
				}
				if text != innoDB {
					code = common.NotInnoDBEngine
					break
				}
			}
		}
	}

	if code != common.Ok {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "InnoDB engine is not used",
			Content: fmt.Sprintf("%q doesn't use InnoDB engine", in.Text()),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *useInnoDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
