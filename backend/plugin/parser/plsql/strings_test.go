package plsql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	schemaName = "BYTEBASE"
)

type stringsManipulateTest struct {
	Input   string
	Actions []testAction
	Want    string
}

type testAction struct {
	Type      string
	Arguments []string
}

func TestStringsManipulate(t *testing.T) {
	tests := []stringsManipulateTest{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_strings_manipulate.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		tree, tokens, err := ParsePLSQL(t.Input)
		a.NoError(err)
		manipulator := NewStringsManipulator(tree, tokens)
		actions := convertActionsForTest(t.Actions)
		result, err := manipulator.Manipulate(actions...)
		a.NoError(err)
		if record {
			tests[i].Want = result
		} else {
			a.Equal(t.Want, result, t.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func convertActionsForTest(actions []testAction) []base.StringsManipulatorAction {
	var result []base.StringsManipulatorAction
	for _, action := range actions {
		switch action.Type {
		case "dropTable":
			result = append(result, NewDropTableAction(schemaName, action.Arguments[0]))
		case "addTable":
			result = append(result, NewAddTableAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "dropColumn":
			result = append(result, NewDropColumnAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "addColumn":
			result = append(result, NewAddColumnAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "modifyColumnType":
			result = append(result, NewModifyColumnTypeAction(schemaName, action.Arguments[0], action.Arguments[1], action.Arguments[2]))
		case "dropColumnOption":
			result = append(result, NewDropColumnOptionAction(schemaName, action.Arguments[0], action.Arguments[1], convertColumnOptionTypeForTest(action.Arguments[2])))
		case "modifyColumnOption":
			result = append(result, NewModifyColumnOptionAction(schemaName, action.Arguments[0], action.Arguments[1], convertColumnOptionTypeForTest(action.Arguments[2]), action.Arguments[3]))
		case "addColumnOption":
			result = append(result, NewAddColumnOptionAction(schemaName, action.Arguments[0], action.Arguments[1], convertColumnOptionTypeForTest(action.Arguments[2]), action.Arguments[3]))
		case "dropTableConstraint":
			result = append(result, NewDropTableConstraintAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "modifyTableConstraint":
			result = append(result, NewModifyTableConstraintAction(schemaName, action.Arguments[0], convertConstraintTypeForTest(action.Arguments[1]), action.Arguments[2], action.Arguments[3]))
		case "addTableConstraint":
			result = append(result, NewAddTableConstraintAction(schemaName, action.Arguments[0], convertConstraintTypeForTest(action.Arguments[1]), action.Arguments[2]))
		case "addIndex":
			result = append(result, NewAddIndexAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "dropIndex":
			result = append(result, NewDropIndexAction(schemaName, action.Arguments[0], action.Arguments[1]))
		case "modifyIndex":
			result = append(result, NewModifyIndexAction(schemaName, action.Arguments[0], action.Arguments[1], action.Arguments[2]))
		}
	}

	return result
}

func convertConstraintTypeForTest(s string) base.TableConstraintType {
	switch s {
	case "primaryKey":
		return base.TableConstraintTypePrimaryKey
	case "unique":
		return base.TableConstraintTypeUnique
	}
	return base.TableConstraintTypeNone
}

func convertColumnOptionTypeForTest(s string) base.ColumnOptionType {
	switch s {
	case "notNull":
		return base.ColumnOptionTypeNotNull
	case "defaultValue":
		return base.ColumnOptionTypeDefault
	}
	return base.ColumnOptionTypeNone
}
