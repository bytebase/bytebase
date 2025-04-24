package mysql

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type splitTestData struct {
	statement string
	want      resData
}

type resData struct {
	res []base.SingleSQL
	err string
}

func generateOneMBInsert() string {
	var rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	letterList := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 1024*1024)
	for i := range b {
		b[i] = letterList[rand.Intn(len(letterList))]
	}
	return fmt.Sprintf("INSERT INTO t values('%s')", string(b))
}

func TestMySQLSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []splitTestData{
		{
			statement: `
			DELIMITER ;;
			CREATE PROCEDURE dorepeat(p1 INT)
			BEGIN
				DECLARE x INT;
				SET x = 0;
				label1: WHILE x < p1 DO
					SET x = x + 1;
				END WHILE label1;
			END;;
			DELIMITER ;
			CALL dorepeat(1000);
			SELECT x;
			`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text: `			CREATE PROCEDURE dorepeat(p1 INT)
			BEGIN
				DECLARE x INT;
				SET x = 0;
				label1: WHILE x < p1 DO
					SET x = x + 1;
				END WHILE label1;
			END;`,
						BaseLine: 2,
						Start:    &storepb.Position{Line: 2, Column: 3},
						End:      &storepb.Position{Line: 9, Column: 7},
					},
					{
						Text:     `			CALL dorepeat(1000);`,
						BaseLine: 11,
						Start:    &storepb.Position{Line: 11, Column: 3},
						End:      &storepb.Position{Line: 11, Column: 22},
					},
					{
						Text: `
			SELECT x;`,
						BaseLine: 11,
						Start:    &storepb.Position{Line: 12, Column: 3},
						End:      &storepb.Position{Line: 12, Column: 11},
					},
					{
						Text:     "\n\t\t\t",
						BaseLine: 12,
						// TODO(zp): Wait, but why the start position is larger than the end position?
						Start: &storepb.Position{Line: 13, Column: 3},
						End:   &storepb.Position{Line: 13, Column: 2},
						Empty: true,
					},
				},
			},
		},
		{
			// 20 IF symbol
			statement: `
			SELECT
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			)
			FROM t; SELECT * FROM t;`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text: `
			SELECT
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			),
			IF (
				age < 18,
				'child',
				'adult'
			)
			FROM t;`,
						Start: &storepb.Position{Line: 1, Column: 3},
						End:   &storepb.Position{Line: 102, Column: 9},
					},
					{
						Text:     " SELECT * FROM t;",
						BaseLine: 102,
						Start:    &storepb.Position{Line: 102, Column: 11},
						End:      &storepb.Position{Line: 102, Column: 26},
					},
				},
			},
		},
		{
			statement: `select * from t;select "\"" where true;`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  `select * from t;`,
						Start: &storepb.Position{Line: 0, Column: 0},
						End:   &storepb.Position{Line: 0, Column: 15},
					},
					{
						Text:  `select "\"" where true;`,
						Start: &storepb.Position{Line: 0, Column: 16},
						End:   &storepb.Position{Line: 0, Column: 38},
					},
				},
			},
		},
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n",
						Start: &storepb.Position{Line: 2, Column: 0},
						End:   &storepb.Position{Line: 1, Column: 22},
						Empty: true,
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  `select * from t;`,
						End:   &storepb.Position{Line: 0, Column: 15},
						Start: &storepb.Position{Line: 0, Column: 0},
					},
					{
						Text: "\n\t\t\t/* sdfasdf */",
						// TODO(zp): Wait, but why the start position is larger than the end position?
						End:   &storepb.Position{Line: 1, Column: 3},
						Start: &storepb.Position{Line: 1, Column: 16},
						Empty: true,
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */;
			select * from t;`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  `select * from t;`,
						Start: &storepb.Position{Line: 0, Column: 0},
						End:   &storepb.Position{Line: 0, Column: 15},
					},
					{
						Text:  "\n\t\t\t/* sdfasdf */;",
						End:   &storepb.Position{Line: 1, Column: 16},
						Start: &storepb.Position{Line: 1, Column: 16},
						Empty: true,
					},
					{
						Text:     "\n\t\t\tselect * from t;",
						BaseLine: 1,
						End:      &storepb.Position{Line: 2, Column: 18},
						Start:    &storepb.Position{Line: 2, Column: 3},
					},
				},
			},
		},
		{
			statement: "CREATE DEFINER=`root`@`%` FUNCTION `CalcIncome`( starting_value INT ) RETURNS int\n" +
				`BEGIN

		   DECLARE income INT;

		   SET income = 0;

		   label1: WHILE income <= 3000 DO
			 SET income = income + starting_value;
		   END WHILE label1;

		   RETURN income;

		END ;`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text: "CREATE DEFINER=`root`@`%` FUNCTION `CalcIncome`( starting_value INT ) RETURNS int\n" +
							`BEGIN

		   DECLARE income INT;

		   SET income = 0;

		   label1: WHILE income <= 3000 DO
			 SET income = income + starting_value;
		   END WHILE label1;

		   RETURN income;

		END ;`,
						Start: &storepb.Position{Line: 0, Column: 0},
						End:   &storepb.Position{Line: 13, Column: 6},
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  bigSQL,
						Start: &storepb.Position{Line: 0, Column: 0},
						End:   &storepb.Position{Line: 0, Column: int32(len(bigSQL) - 1)},
					},
				},
			},
		},
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "    CREATE TABLE t(a int);",
						Start: &storepb.Position{Line: 0, Column: 4},
						End:   &storepb.Position{Line: 0, Column: 25},
					},
					{
						Text:  " CREATE TABLE t1(a int)",
						Start: &storepb.Position{Line: 0, Column: 27},
						End:   &storepb.Position{Line: 0, Column: 48},
					},
				},
			},
		},
		{
			statement: "CREATE TABLE `tech_Book`(id int, name varchar(255));\n" +
				"INSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "CREATE TABLE `tech_Book`(id int, name varchar(255));",
						Start: &storepb.Position{Line: 0, Column: 0},
						End:   &storepb.Position{Line: 0, Column: 51},
					},
					{
						Text:  "\nINSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						Start: &storepb.Position{Line: 1, Column: 0},
						End:   &storepb.Position{Line: 1, Column: 77},
					},
				},
			},
		},
		{
			statement: `
						/* this is the comment. */
						CREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));
						-- this is the comment.
						INSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');
						# this is the comment.
						INSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));",
						Start: &storepb.Position{Line: 2, Column: 6},
						End:   &storepb.Position{Line: 2, Column: 75},
					},
					{
						Text:     "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						Start:    &storepb.Position{Line: 4, Column: 6},
						BaseLine: 2,
						End:      &storepb.Position{Line: 4, Column: 81},
					},
					{
						Text:     "\n\t\t\t\t\t\t# this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						BaseLine: 4,
						Start:    &storepb.Position{Line: 6, Column: 6},
						End:      &storepb.Position{Line: 6, Column: 81},
					},
				},
			},
		},
		{
			statement: `# test for defining stored programs
						CREATE PROCEDURE dorepeat(p1 INT)
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						;
						CALL dorepeat(1000);
						SELECT @x;
						`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						Start: &storepb.Position{Line: 1, Column: 6},
						End:   &storepb.Position{Line: 6, Column: 6},
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine: 6,
						Start:    &storepb.Position{Line: 7, Column: 6},
						End:      &storepb.Position{Line: 7, Column: 25},
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x;",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 8, Column: 6},
						End:      &storepb.Position{Line: 8, Column: 15},
					},
					{
						Text:     "\n\t\t\t\t\t\t",
						BaseLine: 8,
						Start:    &storepb.Position{Line: 9, Column: 6},
						// TODO(zp): Wait, but why the start position is larger than the end position?
						End:   &storepb.Position{Line: 9, Column: 5},
						Empty: true,
					},
				},
			},
		},
		{
			statement: `# test for defining stored programs
						CREATE PROCEDURE dorepeat(p1 INT)
						/* This is a comment */
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						;
						CALL dorepeat(1000);
						SELECT @x;
						`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:  "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\t/* This is a comment */\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						End:   &storepb.Position{Line: 7, Column: 6},
						Start: &storepb.Position{Line: 1, Column: 6},
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 8, Column: 6},
						End:      &storepb.Position{Line: 8, Column: 25},
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x;",
						BaseLine: 8,
						Start:    &storepb.Position{Line: 9, Column: 6},
						End:      &storepb.Position{Line: 9, Column: 15},
					},
					{
						Text:     "\n\t\t\t\t\t\t",
						BaseLine: 9,
						Start:    &storepb.Position{Line: 10, Column: 6},
						End:      &storepb.Position{Line: 10, Column: 5},
						Empty:    true,
					},
				},
			},
		},
		{
			// test for Windows
			statement: `CREATE TABLE t` + "\r\n" + `(a int);` + "\r\n" + `CREATE TABLE t1(b int);`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:     "CREATE TABLE t\r\n(a int);",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 0, Column: 0},
						End:      &storepb.Position{Line: 1, Column: 7},
					},
					{
						Text:     "\r\nCREATE TABLE t1(b int);",
						BaseLine: 1,
						Start:    &storepb.Position{Line: 2, Column: 0},
						End:      &storepb.Position{Line: 2, Column: 22},
					},
				},
			},
		},
	}

	for _, test := range tests {
		res, err := SplitSQL(test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}

func TestSplitMySQLStatements(t *testing.T) {
	tests := []struct {
		statement string
		expected  []string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []string{
				"SELECT * FROM t1 WHERE c1 = 1;",
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END; SELECT * FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END;`,
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END; SELECT REPEAT('123', a) FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END;`,
				" SELECT REPEAT('123', a) FROM t2;",
			},
		},
	}

	for _, test := range tests {
		lexer := parser.NewMySQLLexer(antlr.NewInputStream(test.statement))
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		list, err := splitMySQLStatement(stream, test.statement)
		require.NoError(t, err)
		require.Equal(t, len(test.expected), len(list))
		for i, statement := range list {
			require.Equal(t, test.expected[i], statement.Text)
		}
	}
}
