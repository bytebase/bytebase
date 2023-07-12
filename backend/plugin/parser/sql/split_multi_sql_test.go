package parser

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testData struct {
	statement string
	want      resData
}

type resData struct {
	res []SingleSQL
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

func TestOracleSplitMultiSQL(t *testing.T) {
	tests := []testData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
		},
		{
			statement: `
				select * from t;
				create table table$1 (id int)
			`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t`,
						LastLine: 2,
					},
					{
						Text:     `create table table$1 (id int)`,
						LastLine: 3,
					},
				},
			},
		},
	}

	for _, test := range tests {
		res, err := SplitMultiSQL(Oracle, test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)

		res, err = SplitMultiSQLStream(Oracle, strings.NewReader(test.statement), nil)
		errStr = ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}

func TestPGSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []testData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
		},
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t;`,
						LastLine: 1,
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */;
			select * from t;`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t;`,
						LastLine: 1,
					},
					{
						Text:     `select * from t;`,
						LastLine: 3,
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []SingleSQL{
					{
						Text:     bigSQL,
						LastLine: 1,
					},
				},
			},
		},
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []SingleSQL{
					{
						Text:     "CREATE TABLE t(a int);",
						LastLine: 1,
					},
					{
						Text:     "CREATE TABLE t1(a int)",
						LastLine: 1,
					},
				},
			},
		},
		{
			statement: `CREATE TABLE "tech_Book"(id int, name varchar(255));
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `CREATE TABLE "tech_Book"(id int, name varchar(255));`,
						LastLine: 1,
					},
					{
						Text:     `INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						LastLine: 2,
					},
				},
			},
		},
		{
			statement: `
						/* this is the comment. */
						CREATE /* inline comment */TABLE "tech_Book"(id int, name varchar(255));
						-- this is the comment.
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `/* this is the comment. */
						CREATE /* inline comment */TABLE "tech_Book"(id int, name varchar(255));`,
						LastLine: 3,
					},
					{
						Text: `-- this is the comment.
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						LastLine: 5,
					},
				},
			},
		},
		{
			statement: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdf\'lkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;
						CREATE TABLE t(a int);`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdf\'lkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;`,
						LastLine: 8,
					},
					{
						Text:     `CREATE TABLE t(a int);`,
						LastLine: 9,
					},
				},
			},
		},
		{
			statement: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdf\'lkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;
						CREATE TABLE t(a int);`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdf\'lkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;`,
						LastLine: 8,
					},
					{
						Text:     `CREATE TABLE t(a int);`,
						LastLine: 9,
					},
				},
			},
		},
		{
			// test for Windows
			statement: `CREATE TABLE t` + "\r\n" + `(a int);` + "\r\n" + `CREATE TABLE t1(b int);`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     "CREATE TABLE t\r\n(a int);",
						LastLine: 2,
					},
					{
						Text:     "CREATE TABLE t1(b int);",
						LastLine: 3,
					},
				},
			},
		},
		{
			statement: `INSERT INTO "public"."table"("id","content")
			VALUES
			(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),
			(133,'knuandfan public table id\';create table t(a int, b int);set @text=\'\\\\kdaminxkljasdfiebkla.unkonwn\'+\'abcdef.xyz\\\'; local xxxyy.abcddd.mysql @text;------- '),
			(1444,'table t xyz abc a\'a\\\\\\\\\'b"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\');`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `INSERT INTO "public"."table"("id","content")
			VALUES
			(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),
			(133,'knuandfan public table id\';create table t(a int, b int);set @text=\'\\\\kdaminxkljasdfiebkla.unkonwn\'+\'abcdef.xyz\\\'; local xxxyy.abcddd.mysql @text;------- '),
			(1444,'table t xyz abc a\'a\\\\\\\\\'b"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\');`,
						LastLine: 5,
					},
				},
			},
		},
		{
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				err: "invalid string: not found delimiter: ', but found EOF",
			},
		},
		{
			statement: `INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: "invalid indentifier: not found delimiter: \", but found EOF",
			},
		},
		{
			statement: `/*INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: "invalid comment: not found */, but found EOF",
			},
		},
		{
			statement: `$$INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: "scanTo failed: delimiter \"$$\" not found",
			},
		},
	}

	for _, test := range tests {
		res, err := SplitMultiSQL(Postgres, test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)

		res, err = SplitMultiSQLStream(Postgres, strings.NewReader(test.statement), nil)
		errStr = ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}

func TestMySQLSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []testData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n;",
						LastLine: 3,
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t;`,
						LastLine: 1,
					},
					{
						Text:     "\n\t\t\t/* sdfasdf */\n;",
						LastLine: 3,
					},
				},
			},
		},
		{
			statement: `select * from t;
			/* sdfasdf */;
			select * from t;`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t;`,
						LastLine: 1,
					},
					{
						Text:     "\n\t\t\t/* sdfasdf */;",
						LastLine: 2,
					},
					{
						Text:     "\n\t\t\tselect * from t\n;",
						BaseLine: 1,
						LastLine: 4,
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
				res: []SingleSQL{
					{
						Text: "CREATE DEFINER=`root`@`%` FUNCTION `CalcIncome`( starting_value INT ) RETURNS int\n" +
							`BEGIN

		   DECLARE income INT;

		   SET income = 0;

		   label1: WHILE income <= 3000 DO
			 SET income = income + starting_value;
		   END WHILE label1;

		   RETURN income;

		END
;`,
						LastLine: 15,
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []SingleSQL{
					{
						Text:     bigSQL + "\n;",
						LastLine: 2,
					},
				},
			},
		},
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []SingleSQL{
					{
						Text:     "    CREATE TABLE t(a int);",
						LastLine: 1,
					},
					{
						Text:     " CREATE TABLE t1(a int)\n;",
						LastLine: 2,
					},
				},
			},
		},
		{
			statement: "CREATE TABLE `tech_Book`(id int, name varchar(255));\n" +
				"INSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
			want: resData{
				res: []SingleSQL{
					{
						Text:     "CREATE TABLE `tech_Book`(id int, name varchar(255));",
						LastLine: 1,
					},
					{
						Text:     "\nINSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka')\n;",
						LastLine: 3,
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
				res: []SingleSQL{
					{
						Text:     "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));",
						LastLine: 3,
					},
					{
						Text:     "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						BaseLine: 2,
						LastLine: 5,
					},
					{
						Text:     "\n\t\t\t\t\t\t# this is the comment.\n\t\t\t\t\t\tINSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka')\n;",
						BaseLine: 4,
						LastLine: 8,
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
				res: []SingleSQL{
					{
						Text:     "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						LastLine: 7,
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine: 6,
						LastLine: 8,
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x\n;",
						BaseLine: 7,
						LastLine: 10,
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
				res: []SingleSQL{
					{
						Text:     "# test for defining stored programs\n\t\t\t\t\t\tCREATE PROCEDURE dorepeat(p1 INT)\n\t\t\t\t\t\t/* This is a comment */\n\t\t\t\t\t\tBEGIN\n\t\t\t\t\t\t\tSET @x = 0;\n\t\t\t\t\t\t\tREPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;\n\t\t\t\t\t\tEND\n\t\t\t\t\t\t;",
						LastLine: 8,
					},
					{
						Text:     "\n\t\t\t\t\t\tCALL dorepeat(1000);",
						BaseLine: 7,
						LastLine: 9,
					},
					{
						Text:     "\n\t\t\t\t\t\tSELECT @x\n;",
						BaseLine: 8,
						LastLine: 11,
					},
				},
			},
		},
		{
			// test for Windows
			statement: `CREATE TABLE t` + "\r\n" + `(a int);` + "\r\n" + `CREATE TABLE t1(b int);`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     "CREATE TABLE t\r\n(a int);",
						LastLine: 2,
					},
					{
						Text:     "\r\nCREATE TABLE t1(b int)\n;",
						BaseLine: 1,
						LastLine: 4,
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
		res, err := SplitMultiSQL(MySQL, test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)

		res, err = SplitMultiSQLStream(MySQL, strings.NewReader(test.statement), nil)
		errStr = ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}

func TestMySQLSplitMultiSQLAndNormalize(t *testing.T) {
	tests := []testData{
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     `select * from t;`,
						LastLine: 1,
					},
					{
						Text:     "\n\t\t\t/* sdfasdf */\n;",
						LastLine: 3,
					},
				},
			},
		},
		{
			statement: `
			DROP PROCEDURE IF EXISTS p1;
			
			CREATE PROCEDURE p1()
			BEGIN
				SELECT count(*) from t;
			END;

			DROP PROCEDURE IF EXISTS p2;
			
			CREATE PROCEDURE p2()
			BEGIN
				SELECT count(*) from t;
			END;
			`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     "\n\t\t\tDROP PROCEDURE IF EXISTS p1;",
						LastLine: 2,
					},
					{
						Text:     "\n\t\t\t\n\t\t\tCREATE PROCEDURE p1()\n\t\t\tBEGIN\n\t\t\t\tSELECT count(*) from t;\n\t\t\tEND;",
						BaseLine: 1,
						LastLine: 7,
					},
					{
						Text:     "\n\n\t\t\tDROP PROCEDURE IF EXISTS p2;",
						BaseLine: 6,
						LastLine: 9,
					},
					{
						Text:     "\n\t\t\t\n\t\t\tCREATE PROCEDURE p2()\n\t\t\tBEGIN\n\t\t\t\tSELECT count(*) from t;\n\t\t\tEND\n;",
						BaseLine: 8,
						LastLine: 15,
					},
				},
			},
		},
	}
	for _, test := range tests {
		res, err := SplitMultiSQLAndNormalize(MySQL, test.statement)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}
