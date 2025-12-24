package mysql

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/mysql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type splitTestData struct {
	statement string
	want      resData
}

type resData struct {
	res []base.Statement
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
				res: []base.Statement{
					{
						Text: `			CREATE PROCEDURE dorepeat(p1 INT)
			BEGIN
				DECLARE x INT;
				SET x = 0;
				label1: WHILE x < p1 DO
					SET x = x + 1;
				END WHILE label1;
			END;`,
						Start:    &storepb.Position{Line: 3, Column: 4},
						End:      &storepb.Position{Line: 10, Column: 8},
						Range:    &storepb.Range{Start: 17, End: 175},
					},
					{
						Text:     `			CALL dorepeat(1000);`,
						Start:    &storepb.Position{Line: 12, Column: 4},
						End:      &storepb.Position{Line: 12, Column: 23},
						Range:    &storepb.Range{Start: 191, End: 214},
					},
					{
						Text: `
			SELECT x;`,
						Start:    &storepb.Position{Line: 13, Column: 4},
						End:      &storepb.Position{Line: 13, Column: 12},
						Range:    &storepb.Range{Start: 214, End: 227},
					},
					{
						Text:     "\n\t\t\t",
						// TODO(zp): Wait, but why the start position is larger than the end position?
						Start: &storepb.Position{Line: 14, Column: 4},
						End:   &storepb.Position{Line: 14, Column: 3},
						Empty: true,
						Range: &storepb.Range{Start: 227, End: 231},
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
				res: []base.Statement{
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
						Start: &storepb.Position{Line: 2, Column: 4},
						End:   &storepb.Position{Line: 103, Column: 10},
						Range: &storepb.Range{Start: 0, End: 1080},
					},
					{
						Text:     " SELECT * FROM t;",
						Start:    &storepb.Position{Line: 103, Column: 12},
						End:      &storepb.Position{Line: 103, Column: 27},
						Range:    &storepb.Range{Start: 1080, End: 1097},
					},
				},
			},
		},
		{
			statement: `select * from t;select "\"" where true;`,
			want: resData{
				res: []base.Statement{
					{
						Text:  `select * from t;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 16},
						Range: &storepb.Range{Start: 0, End: 16},
					},
					{
						Text:  `select "\"" where true;`,
						Start: &storepb.Position{Line: 1, Column: 17},
						End:   &storepb.Position{Line: 1, Column: 39},
						Range: &storepb.Range{Start: 16, End: 39},
					},
				},
			},
		},
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
`,
			want: resData{
				res: []base.Statement{
					{
						Text:  "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n",
						Start: &storepb.Position{Line: 3, Column: 1},
						End:   &storepb.Position{Line: 2, Column: 23},
						Empty: true,
						Range: &storepb.Range{Start: 0, End: 39},
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []base.Statement{
					{
						Text:  `select * from t;`,
						End:   &storepb.Position{Line: 1, Column: 16},
						Start: &storepb.Position{Line: 1, Column: 1},
						Range: &storepb.Range{Start: 0, End: 16},
					},
					{
						Text: "\n\t\t\t/* sdfasdf */",
						// TODO(zp): Wait, but why the start position is larger than the end position?
						End:   &storepb.Position{Line: 2, Column: 4},
						Start: &storepb.Position{Line: 2, Column: 17},
						Empty: true,
						Range: &storepb.Range{Start: 16, End: 33},
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */;
			select * from t;`,
			want: resData{
				res: []base.Statement{
					{
						Text:  `select * from t;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 16},
						Range: &storepb.Range{Start: 0, End: 16},
					},
					{
						Text:  "\n\t\t\t/* sdfasdf */;",
						End:   &storepb.Position{Line: 2, Column: 17},
						Start: &storepb.Position{Line: 2, Column: 17},
						Empty: true,
						Range: &storepb.Range{Start: 16, End: 34},
					},
					{
						Text:     "\n\t\t\tselect * from t;",
						End:      &storepb.Position{Line: 3, Column: 19},
						Start:    &storepb.Position{Line: 3, Column: 4},
						Range:    &storepb.Range{Start: 34, End: 54},
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
				res: []base.Statement{
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
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 14, Column: 7},
						Range: &storepb.Range{Start: 0, End: 268},
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []base.Statement{
					{
						Text:  bigSQL,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: int32(len(bigSQL))},
						Range: &storepb.Range{Start: 0, End: int32(len(bigSQL))},
					},
				},
			},
		},
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []base.Statement{
					{
						Text:  "    CREATE TABLE t(a int);",
						Start: &storepb.Position{Line: 1, Column: 5},
						End:   &storepb.Position{Line: 1, Column: 26},
						Range: &storepb.Range{Start: 0, End: 26},
					},
					{
						Text:  " CREATE TABLE t1(a int)",
						Start: &storepb.Position{Line: 1, Column: 28},
						End:   &storepb.Position{Line: 1, Column: 49},
						Range: &storepb.Range{Start: 26, End: 49},
					},
				},
			},
		},
		{
			statement: "CREATE TABLE `tech_Book`(id int, name varchar(255));\n" +
				"INSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
			want: resData{
				res: []base.Statement{
					{
						Text:  "CREATE TABLE `tech_Book`(id int, name varchar(255));",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 52},
						Range: &storepb.Range{Start: 0, End: 52},
					},
					{
						Text:  "\nINSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						Start: &storepb.Position{Line: 2, Column: 1},
						End:   &storepb.Position{Line: 2, Column: 78},
						Range: &storepb.Range{Start: 52, End: 131},
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
				res: []base.Statement{
					{
						Text:  "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));",
						Start: &storepb.Position{Line: 3, Column: 7},
						End:   &storepb.Position{Line: 3, Column: 76},
						Range: &storepb.Range{Start: 0, End: 110},
					},
					{
						Text:     "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						Start:    &storepb.Position{Line: 5, Column: 7},
						End:      &storepb.Position{Line: 5, Column: 82},
						Range:    &storepb.Range{Start: 110, End: 223},
					},
					{
						Text:     "\n\t\t\t\t\t\t# this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						Start:    &storepb.Position{Line: 7, Column: 7},
						End:      &storepb.Position{Line: 7, Column: 82},
						Range:    &storepb.Range{Start: 223, End: 335},
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
				res: []base.Statement{
					{
						Text:  "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						Start: &storepb.Position{Line: 2, Column: 7},
						End:   &storepb.Position{Line: 7, Column: 7},
						Range: &storepb.Range{Start: 0, End: 181},
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						Start:    &storepb.Position{Line: 8, Column: 7},
						End:      &storepb.Position{Line: 8, Column: 26},
						Range:    &storepb.Range{Start: 181, End: 208},
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x;",
						Start:    &storepb.Position{Line: 9, Column: 7},
						End:      &storepb.Position{Line: 9, Column: 16},
						Range:    &storepb.Range{Start: 208, End: 225},
					},
					{
						Text:     "\n\t\t\t\t\t\t",
						Start:    &storepb.Position{Line: 10, Column: 7},
						// TODO(zp): Wait, but why the start position is larger than the end position?
						End:   &storepb.Position{Line: 10, Column: 6},
						Empty: true,
						Range: &storepb.Range{Start: 225, End: 232},
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
				res: []base.Statement{
					{
						Text:  "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\t/* This is a comment */\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						End:   &storepb.Position{Line: 8, Column: 7},
						Start: &storepb.Position{Line: 2, Column: 7},
						Range: &storepb.Range{Start: 0, End: 211},
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						Start:    &storepb.Position{Line: 9, Column: 7},
						End:      &storepb.Position{Line: 9, Column: 26},
						Range:    &storepb.Range{Start: 211, End: 238},
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x;",
						Start:    &storepb.Position{Line: 10, Column: 7},
						End:      &storepb.Position{Line: 10, Column: 16},
						Range:    &storepb.Range{Start: 238, End: 255},
					},
					{
						Text:     "\n\t\t\t\t\t\t",
						Start:    &storepb.Position{Line: 11, Column: 7},
						End:      &storepb.Position{Line: 11, Column: 6},
						Empty:    true,
						Range:    &storepb.Range{Start: 255, End: 262},
					},
				},
			},
		},
		{
			// test for Windows
			statement: `CREATE TABLE t` + "\r\n" + `(a int);` + "\r\n" + `CREATE TABLE t1(b int);`,
			want: resData{
				res: []base.Statement{
					{
						Text:     "CREATE TABLE t\r\n(a int);",
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 2, Column: 8},
						Range:    &storepb.Range{Start: 0, End: 24},
					},
					{
						Text:     "\r\nCREATE TABLE t1(b int);",
						Start:    &storepb.Position{Line: 3, Column: 1},
						End:      &storepb.Position{Line: 3, Column: 23},
						Range:    &storepb.Range{Start: 24, End: 49},
					},
				},
			},
		},
		{
			statement: "SELECT * FROM 表名; INSERT INTO 表 VALUES (1);",
			want: resData{
				res: []base.Statement{
					{
						Text:  "SELECT * FROM 表名;",
						Range: &storepb.Range{Start: 0, End: 21}, // Byte offset 0-21 (not 0-17)
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 17},
					},
					{
						Text:  " INSERT INTO 表 VALUES (1);",
						Range: &storepb.Range{Start: 21, End: 49}, // Byte offset 21-49 (not 17-43)
						Start: &storepb.Position{Line: 1, Column: 19},
						End:   &storepb.Position{Line: 1, Column: 43},
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
