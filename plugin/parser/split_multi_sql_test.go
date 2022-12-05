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
	rand.Seed(time.Now().UnixNano())
	letterList := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 1024*1024)
	for i := range b {
		b[i] = letterList[rand.Intn(len(letterList))]
	}
	return fmt.Sprintf("INSERT INTO t values('%s')", string(b))
}

func TestPGSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []testData{
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
			statement: "DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` FUNCTION `CalcIncome`( starting_value INT ) RETURNS int\n" +
				`BEGIN

		   DECLARE income INT;

		   SET income = 0;

		   label1: WHILE income <= 3000 DO
			 SET income = income + starting_value;
		   END WHILE label1;

		   RETURN income;

		END ;;
		DELIMITER ;`,
			want: resData{
				res: []SingleSQL{
					{
						Text:     "DELIMITER ;;",
						LastLine: 1,
					},
					{
						Text: "CREATE DEFINER=`root`@`%` FUNCTION `CalcIncome`( starting_value INT ) RETURNS int\n" +
							`BEGIN

		   DECLARE income INT;

		   SET income = 0;

		   label1: WHILE income <= 3000 DO
			 SET income = income + starting_value;
		   END WHILE label1;

		   RETURN income;

		END ;;`,
						LastLine: 15,
					},
					{
						Text:     "DELIMITER ;",
						LastLine: 16,
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
			statement: "CREATE TABLE `tech_Book`(id int, name varchar(255));\n" +
				"INSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
			want: resData{
				res: []SingleSQL{
					{
						Text:     "CREATE TABLE `tech_Book`(id int, name varchar(255));",
						LastLine: 1,
					},
					{
						Text:     "INSERT INTO `tech_Book` VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\\'jdfl;\"ka');",
						LastLine: 2,
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
						Text: `/* this is the comment. */
						CREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));`,
						LastLine: 3,
					},
					{
						Text: `-- this is the comment.
						INSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						LastLine: 5,
					},
					{
						Text: `# this is the comment.
						INSERT INTO tech_Book VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						LastLine: 7,
					},
				},
			},
		},
		{
			statement: `# test for defining stored programs
						delimiter //
						CREATE PROCEDURE dorepeat(p1 INT)
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						//
						delimiter ;
						CALL dorepeat(1000);
						SELECT @x;
						`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `# test for defining stored programs
						delimiter //`,
						LastLine: 2,
					},
					{
						Text: `CREATE PROCEDURE dorepeat(p1 INT)
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						//`,
						LastLine: 8,
					},
					{
						Text:     `delimiter ;`,
						LastLine: 9,
					},
					{
						Text:     `CALL dorepeat(1000);`,
						LastLine: 10,
					},
					{
						Text:     `SELECT @x;`,
						LastLine: 11,
					},
				},
			},
		},
		{
			statement: `# test for defining stored programs
						delimiter //
						CREATE PROCEDURE dorepeat(p1 INT)
						/* This is a comment */
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						//
						delimiter ;
						CALL dorepeat(1000);
						SELECT @x;
						`,
			want: resData{
				res: []SingleSQL{
					{
						Text: `# test for defining stored programs
						delimiter //`,
						LastLine: 2,
					},
					{
						Text: `CREATE PROCEDURE dorepeat(p1 INT)
						/* This is a comment */
						BEGIN
							SET @x = 0;
							REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
						END
						//`,
						LastLine: 9,
					},
					{
						Text:     `delimiter ;`,
						LastLine: 10,
					},
					{
						Text:     `CALL dorepeat(1000);`,
						LastLine: 11,
					},
					{
						Text:     `SELECT @x;`,
						LastLine: 12,
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
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				err: "invalid string: not found delimiter: ', but found EOF",
			},
		},
		{
			statement: "INSERT INTO `t VALUES ('klajfas)",
			want: resData{
				err: "invalid indentifier: not found delimiter: `, but found EOF",
			},
		},
		{
			statement: "/*INSERT INTO `t VALUES ('klajfas)",
			want: resData{
				err: "invalid comment: not found */, but found EOF",
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
