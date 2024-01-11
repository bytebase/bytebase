package tidb

import (
	"io"
	"os"
	"testing"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
		manipulator := NewStringsManipulator(t.Input)
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

func convertActionsForTest(actions []testAction) []StringsManipulatorAction {
	var result []StringsManipulatorAction
	for _, action := range actions {
		switch action.Type {
		case "dropTable":
			result = append(result, NewDropTableAction(action.Arguments[0]))
		case "addTable":
			result = append(result, NewAddTableAction(action.Arguments[0]))
		case "dropColumn":
			result = append(result, NewDropColumnAction(action.Arguments[0], action.Arguments[1]))
		case "addColumn":
			result = append(result, NewAddColumnAction(action.Arguments[0], action.Arguments[1]))
		case "modifyColumnType":
			result = append(result, NewModifyColumnTypeAction(action.Arguments[0], action.Arguments[1], action.Arguments[2]))
		case "dropColumnOption":
			result = append(result, NewDropColumnOptionAction(action.Arguments[0], action.Arguments[1], convertColumnOptionTypeForTest(action.Arguments[2])))
		case "modifyColumnOption":
			result = append(result, NewModifyColumnOptionAction(action.Arguments[0], action.Arguments[1], convertColumnOptionTypeForTest(action.Arguments[2]), action.Arguments[3]))
		case "addColumnOption":
			result = append(result, NewAddColumnOptionAction(action.Arguments[0], action.Arguments[1], action.Arguments[2]))
		case "dropTableConstraint":
			result = append(result, NewDropTableConstraintAction(action.Arguments[0], action.Arguments[1]))
		case "modifyTableConstraint":
			result = append(result, NewModifyTableConstraintAction(action.Arguments[0], convertConstraintTypeForTest(action.Arguments[1]), action.Arguments[2], action.Arguments[3]))
		case "addTableConstraint":
			result = append(result, NewAddTableConstraintAction(action.Arguments[0], convertConstraintTypeForTest(action.Arguments[1]), action.Arguments[2]))
		case "dropTableOption":
			result = append(result, NewDropTableOptionAction(action.Arguments[0], convertTableOptionForTest(action.Arguments[1])))
		case "modifyTableOption":
			result = append(result, NewModifyTableOptionAction(action.Arguments[0], convertTableOptionForTest(action.Arguments[1]), action.Arguments[2]))
		case "addTableOption":
			result = append(result, NewAddTableOptionAction(action.Arguments[0], action.Arguments[1]))
		}
	}

	return result
}

func convertTableOptionForTest(s string) tidbast.TableOptionType {
	switch s {
	case "comment":
		return tidbast.TableOptionComment
	default:
		return tidbast.TableOptionNone
	}
}

func convertConstraintTypeForTest(s string) tidbast.ConstraintType {
	switch s {
	case "primaryKey":
		return tidbast.ConstraintPrimaryKey
	case "key":
		return tidbast.ConstraintKey
	case "index":
		return tidbast.ConstraintIndex
	case "uniqKey":
		return tidbast.ConstraintUniqKey
	case "uniqIndex":
	case "foreignKey":
		return tidbast.ConstraintForeignKey
	case "unique":
		return tidbast.ConstraintUniq
	case "check":
		return tidbast.ConstraintCheck
	case "fulltext":
		return tidbast.ConstraintFulltext
	}
	return tidbast.ConstraintNoConstraint
}

func convertColumnOptionTypeForTest(s string) tidbast.ColumnOptionType {
	switch s {
	case "primaryKey":
		return tidbast.ColumnOptionPrimaryKey
	case "notNull":
		return tidbast.ColumnOptionNotNull
	case "autoIncrement":
		return tidbast.ColumnOptionAutoIncrement
	case "defaultValue":
		return tidbast.ColumnOptionDefaultValue
	case "uniqKey":
		return tidbast.ColumnOptionUniqKey
	case "null":
		return tidbast.ColumnOptionNull
	case "onUpdate":
		return tidbast.ColumnOptionOnUpdate
	case "fulltext":
		return tidbast.ColumnOptionFulltext
	case "comment":
		return tidbast.ColumnOptionComment
	case "generated":
		return tidbast.ColumnOptionGenerated
	case "reference":
		return tidbast.ColumnOptionReference
	case "collate":
		return tidbast.ColumnOptionCollate
	case "check":
		return tidbast.ColumnOptionCheck
	case "columnFormat":
		return tidbast.ColumnOptionColumnFormat
	case "storage":
		return tidbast.ColumnOptionStorage
	case "autoRandom":
		return tidbast.ColumnOptionAutoRandom
	}
	return tidbast.ColumnOptionNoOption
}
