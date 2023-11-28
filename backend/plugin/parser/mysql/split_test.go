package mysql

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n",
						FirstStatementLine:   1,
						FirstStatementColumn: 22,
						LastLine:             1,
						LastColumn:           22,
						Empty:                true,
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
						Text:                 `select * from t;`,
						LastLine:             0,
						LastColumn:           15,
						FirstStatementLine:   0,
						FirstStatementColumn: 0,
					},
					{
						Text:                 "\n\t\t\t/* sdfasdf */",
						LastLine:             1,
						LastColumn:           3,
						FirstStatementLine:   1,
						FirstStatementColumn: 3,
						Empty:                true,
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
						Text:       `select * from t;`,
						LastLine:   0,
						LastColumn: 15,
					},
					{
						Text:                 "\n\t\t\t/* sdfasdf */;",
						LastLine:             1,
						LastColumn:           16,
						FirstStatementLine:   1,
						FirstStatementColumn: 16,
						Empty:                true,
					},
					{
						Text:                 "\n\t\t\tselect * from t;",
						BaseLine:             1,
						LastLine:             2,
						LastColumn:           18,
						FirstStatementLine:   2,
						FirstStatementColumn: 3,
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
						LastLine:   13,
						LastColumn: 6,
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       bigSQL + ";",
						LastLine:   0,
						LastColumn: len(bigSQL),
					},
				},
			},
		},
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "    CREATE TABLE t(a int);",
						LastLine:             0,
						LastColumn:           25,
						FirstStatementColumn: 4,
					},
					{
						Text:                 " CREATE TABLE t1(a int);",
						LastLine:             0,
						LastColumn:           49,
						FirstStatementColumn: 27,
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
						Text:       "CREATE TABLE `tech_Book`(id int, name varchar(255));",
						LastLine:   0,
						LastColumn: 51,
					},
					{
						Text:               "\nINSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						LastLine:           1,
						LastColumn:         77,
						FirstStatementLine: 1,
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
						Text:                 "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));",
						LastLine:             2,
						LastColumn:           75,
						FirstStatementLine:   2,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						BaseLine:             2,
						LastLine:             4,
						LastColumn:           81,
						FirstStatementLine:   4,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\t# this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						BaseLine:             4,
						LastLine:             6,
						LastColumn:           81,
						FirstStatementLine:   6,
						FirstStatementColumn: 6,
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
						Text:                 "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						LastLine:             6,
						LastColumn:           6,
						FirstStatementLine:   1,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine:             6,
						LastLine:             7,
						LastColumn:           25,
						FirstStatementLine:   7,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\tSELECT @x;",
						BaseLine:             7,
						LastLine:             8,
						LastColumn:           15,
						FirstStatementLine:   8,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\t",
						BaseLine:             8,
						LastLine:             9,
						LastColumn:           5,
						FirstStatementLine:   9,
						FirstStatementColumn: 5,
						Empty:                true,
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
						Text:                 "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\t/* This is a comment */\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						FirstStatementLine:   1,
						FirstStatementColumn: 6,
						LastLine:             7,
						LastColumn:           6,
					},
					{
						Text:                 "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine:             7,
						LastLine:             8,
						LastColumn:           25,
						FirstStatementLine:   8,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\tSELECT @x;",
						BaseLine:             8,
						LastLine:             9,
						LastColumn:           15,
						FirstStatementLine:   9,
						FirstStatementColumn: 6,
					},
					{
						Text:                 "\n\t\t\t\t\t\t",
						BaseLine:             9,
						LastLine:             10,
						LastColumn:           5,
						FirstStatementLine:   10,
						FirstStatementColumn: 5,
						Empty:                true,
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
						Text:       "CREATE TABLE t\r\n(a int);",
						LastLine:   1,
						LastColumn: 7,
					},
					{
						Text:                 "\r\nCREATE TABLE t1(b int);",
						BaseLine:             1,
						LastLine:             2,
						LastColumn:           22,
						FirstStatementLine:   2,
						FirstStatementColumn: 0,
					},
				},
			},
		},
		{
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				err: "failed to split multi sql: invalid string: not found delimiter: ', but found EOF",
			},
		},
		{
			statement: "INSERT INTO `t VALUES ('klajfas)",
			want: resData{
				err: "failed to split multi sql: invalid indentifier: not found delimiter: `, but found EOF",
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

		res, err = SplitMultiSQLStream(strings.NewReader(test.statement), nil)
		errStr = ""
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
		list, err := splitMySQLStatement(stream)
		require.NoError(t, err)
		require.Equal(t, len(test.expected), len(list))
		for i, statement := range list {
			require.Equal(t, test.expected[i], statement.Text)
		}
	}
}
