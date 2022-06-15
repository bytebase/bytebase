package pg

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
	"github.com/stretchr/testify/require"
)

type testData struct {
	statement string
	want      string
}

func TestPostgreSQLParser(t *testing.T) {
	type walkerContext struct {
		output string
	}
	tests := []testData{
		{
			statement: "DROP DATABASE d1;",
			want:      "Drop database d1",
		},
		{
			statement: "DROP TABLE t1;",
			want:      "Drop table t1",
		},
		{
			statement: "DROP VIEW v1;",
			want:      "Drop view v1",
		},
		{
			statement: "CREATE UNIQUE INDEX idx1 ON t1(f1);",
			want:      "Create unique index idx1 on t1(f1)",
		},
		{
			statement: "ALTER TABLE t1 RENAME TO t2;",
			want:      "Alter table t1 rename to t2",
		},
		{
			statement: "ALTER TABLE t1 ADD f1 TEXT;",
			want:      "It's *tree.AlterTableAddColumn for Table t1. Add Column f1",
		},
		{
			statement: "ALTER TABLE t1 DROP COLUMN f1;",
			want:      "It's *tree.AlterTableDropColumn for Table t1. Drop Column f1",
		},
		{
			statement: "ALTER TABLE t1 ADD CONSTRAINT pk_t PRIMARY KEY (f1);",
			want:      "It's *tree.AlterTableAddConstraint for Table t1. Add primary key constraint pk_t: f1",
		},
		{
			statement: "ALTER TABLE t1 ADD CONSTRAINT uk_t UNIQUE (f);",
			want:      "It's *tree.AlterTableAddConstraint for Table t1. Add unique constraint uk_t: f",
		},
		{
			statement: "ALTER TABLE t1 ADD FOREIGN KEY(f1) REFERENCES t2(f2);",
			want:      "It's *tree.AlterTableAddConstraint for Table t1. Add foreign key constraint : (f1) references (f2)",
		},
		{
			statement: "ALTER TABLE t1 ADD CHECK (f1 > 0);",
			want:      "It's *tree.AlterTableAddConstraint for Table t1. Add check constraint : f1 > 0",
		},
	}

	w := &walk.AstWalker{
		Fn: func(ctx interface{}, node interface{}) (stop bool) {
			var buffer bytes.Buffer
			wc, ok := ctx.(*walkerContext)
			if !ok {
				return true
			}
			switch p := node.(type) {
			case *tree.DropDatabase:
				buffer.WriteString(fmt.Sprintf("Drop database %s", p.Name))
			case *tree.DropTable:
				var nameList []string
				for _, name := range p.Names {
					nameList = append(nameList, name.Table())
				}
				buffer.WriteString(fmt.Sprintf("Drop table %s", strings.Join(nameList, ", ")))
			case *tree.DropView:
				var nameList []string
				for _, name := range p.Names {
					nameList = append(nameList, name.Table())
				}
				buffer.WriteString(fmt.Sprintf("Drop view %s", strings.Join(nameList, ", ")))
			case *tree.CreateIndex:
				buffer.WriteString("Create ")
				if p.Unique {
					buffer.WriteString("unique ")
				}
				var nameList []string
				for _, name := range p.Columns {
					nameList = append(nameList, name.Column.String())
				}
				buffer.WriteString(fmt.Sprintf("index %s on %s(%s)", p.Name, p.Table.TableName, strings.Join(nameList, ", ")))
			case *tree.RenameTable:
				buffer.WriteString("Alter ")
				if p.IsSequence {
					buffer.WriteString("sequence ")
				} else if p.IsView {
					buffer.WriteString("view ")
				} else {
					buffer.WriteString("table ")
				}
				buffer.WriteString(fmt.Sprintf("%s rename to %s", p.Name, p.NewName))
			case *tree.AlterTable:
				for _, cmd := range p.Cmds {
					buffer.WriteString(fmt.Sprintf("It's %T for Table %s. ", cmd, p.Table.ToTableName().TableName))
					switch v := cmd.(type) {
					case *tree.AlterTableRenameTable:
						buffer.WriteString(fmt.Sprintf("Rename to %s", v.NewName.Table()))
					case *tree.AlterTableAddColumn:
						buffer.WriteString(fmt.Sprintf("Add Column %s", v.ColumnDef.Name))
					case *tree.AlterTableDropColumn:
						buffer.WriteString(fmt.Sprintf("Drop Column %s", v.Column))
					case *tree.AlterTableAddConstraint:
						switch cons := v.ConstraintDef.(type) {
						case *tree.CheckConstraintTableDef:
							buffer.WriteString(fmt.Sprintf("Add check constraint %s: %s", cons.Name, cons.Expr))
						case *tree.UniqueConstraintTableDef:
							var colNames []string
							for _, col := range cons.Columns {
								colNames = append(colNames, col.Column.String())
							}
							if cons.PrimaryKey {
								buffer.WriteString(fmt.Sprintf("Add primary key constraint %s: %s", cons.Name, strings.Join(colNames, ", ")))
							} else {
								buffer.WriteString(fmt.Sprintf("Add unique constraint %s: %s", cons.Name, strings.Join(colNames, ", ")))
							}
						case *tree.ForeignKeyConstraintTableDef:
							var fromList, toList []string
							for _, col := range cons.FromCols {
								fromList = append(fromList, col.String())
							}
							for _, col := range cons.ToCols {
								toList = append(toList, col.String())
							}
							buffer.WriteString(fmt.Sprintf("Add foreign key constraint %s: (%s) references (%s)", cons.Name, strings.Join(fromList, ", "), strings.Join(toList, ", ")))
						}
					}
				}
			}
			wc.output = buffer.String()
			return false
		},
	}

	for _, test := range tests {
		stmts, err := parser.Parse(test.statement)
		require.NoError(t, err)
		require.Len(t, stmts, 1)

		ctx := walkerContext{}
		ok, err := w.Walk(stmts, &ctx)
		require.True(t, ok)
		require.NoError(t, err)

		require.Equal(t, test.want, ctx.output)
	}
}
