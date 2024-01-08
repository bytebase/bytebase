package tidb

import (
	"bufio"
	"regexp"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"
)

type StringsManipulator struct {
	s string
}

func NewStringsManipulator(s string) *StringsManipulator {
	return &StringsManipulator{s}
}

type StringsManipulatorAction interface {
	getTopLevelNaming() string
	getSecondLevelNaming() string
}

type StringsManipulatorActionDropTable struct {
	Table string
}

func (s *StringsManipulatorActionDropTable) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropTable) getSecondLevelNaming() string {
	return ""
}

type StringsManipulatorActionDropColumn struct {
	Table  string
	Column string
}

func (s *StringsManipulatorActionDropColumn) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumn) getSecondLevelNaming() string {
	return s.Column
}

type StringsManipulatorActionModifyColumnType struct {
	Table  string
	Column string
	Type   string
}

func (s *StringsManipulatorActionModifyColumnType) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyColumnType) getSecondLevelNaming() string {
	return s.Column
}

type StringsManipulatorActionDropColumnOption struct {
	Table  string
	Column string
	Option tidbast.ColumnOptionType
}

func (s *StringsManipulatorActionDropColumnOption) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumnOption) getSecondLevelNaming() string {
	return s.Column
}

type StringsManipulatorActionModifyColumnOption struct {
	Table           string
	Column          string
	OldOption       tidbast.ColumnOptionType
	NewOptionDefine string
}

func (s *StringsManipulatorActionModifyColumnOption) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyColumnOption) getSecondLevelNaming() string {
	return s.Column
}

type StringsManipulatorActionDropTableConstraint struct {
	Table          string
	Constraint     tidbast.ConstraintType
	ConstraintName string
}

func (s *StringsManipulatorActionDropTableConstraint) getTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropTableConstraint) getSecondLevelNaming() string {
	return s.ConstraintName
}

type StringsManipulatorActionModifyTableConstraint struct {
	Table               string
	OldConstraint       tidbast.ConstraintType
	OldConstraintName   string
	NewConstraintDefine string
}

func (s *StringsManipulatorActionModifyTableConstraint) getTopLevelNaming() string {
	return s.Table
}

var (
	regexpColumn = regexp.MustCompile("^  `([^`]+)`")
)

func (s *StringsManipulator) Manipulate(actions ...StringsManipulatorAction) (string, error) {
	tableActions := make(map[string][]StringsManipulatorAction)

	for _, action := range actions {
		tableName := action.getTopLevelNaming()
		// do copy
		action := action
		tableActions[tableName] = append(tableActions[tableName], action)
	}

	stmts, err := SplitSQL(s.s)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	var results []string

	for _, stmt := range stmts {
		if stmt.Empty {
			results = append(results, stmt.Text)
			continue
		}
		isCreateTable, tableName := extractTableNameForCreateTable(stmt.Text)
		if !isCreateTable {
			results = append(results, stmt.Text)
			continue
		}
		actions, ok := tableActions[tableName]
		if !ok || len(actions) == 0 {
			results = append(results, stmt.Text)
			continue
		}

		var tableActions []StringsManipulatorAction
		actionsMap := make(map[string][]StringsManipulatorAction)
		for _, action := range actions {
			// do copy
			action := action
			secondName := action.getSecondLevelNaming()
			if secondName == "" {
				tableActions = append(tableActions, action)
			} else {
				actionsMap[secondName] = append(actionsMap[secondName], action)
			}
		}

		scanner := bufio.NewScanner(strings.NewReader(stmt.Text))
		for scanner.Scan() {
			line := scanner.Text()

			columnMatch := regexpColumn.FindStringSubmatch(line)
			if len(columnMatch) > 1 {
				// is column definition
				columnName := columnMatch[1]
				actions, ok := actionsMap[columnName]
				if !ok || len(actions) == 0 {
					results = append(results, line)
					continue
				}

			}
		}
		if err := scanner.Err(); err != nil {
			return "", errors.Wrap(err, "failed to scan create table statement")
		}
	}

	return strings.Join(results, ""), nil
}

var (
	regexpPattern = regexp.MustCompile("(?m)^-- Table structure for `([^`]+)`")
)

func extractTableNameForCreateTable(s string) (bool, string) {
	matches := regexpPattern.FindStringSubmatch(s)
	if len(matches) > 1 {
		return true, matches[1]
	}
	return false, ""
}
