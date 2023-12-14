package pg

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

func TestPGSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []splitTestData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n\t\t\t",
						BaseLine:             0,
						FirstStatementLine:   2,
						FirstStatementColumn: 0,
						LastLine:             2,
						LastColumn:           0,
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
						Text:       `select * from t;`,
						LastLine:   0,
						LastColumn: 15,
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
						FirstStatementLine:   1,
						FirstStatementColumn: 3,
						LastLine:             1,
						LastColumn:           16,
						Empty:                true,
					},
					{
						Text:                 "\n\t\t\tselect * from t;",
						BaseLine:             1,
						FirstStatementLine:   2,
						FirstStatementColumn: 3,
						LastLine:             2,
						LastColumn:           18,
					},
				},
			},
		},
		{
			statement: bigSQL,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       bigSQL,
						LastLine:   0,
						LastColumn: len(bigSQL) - 1,
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
						FirstStatementColumn: 4,
						LastLine:             0,
						LastColumn:           25,
					},
					{
						Text:                 " CREATE TABLE t1(a int)",
						FirstStatementColumn: 27,
						LastColumn:           48,
					},
				},
			},
		},
		{
			statement: `CREATE TABLE "tech_Book"(id int, name varchar(255));
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;"ka');`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       `CREATE TABLE "tech_Book"(id int, name varchar(255));`,
						LastLine:   0,
						LastColumn: 51,
					},
					{
						Text:                 "\n\t\t\t\t\t\tINSERT INTO \"tech_Book\" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;\"ka');",
						FirstStatementLine:   1,
						FirstStatementColumn: 6,
						LastLine:             1,
						LastColumn:           81,
					},
				},
			},
		},
		{
			statement: `
						/* this is the comment. */
						CREATE /* inline comment */TABLE "tech_Book"(id int, name varchar(255));
						-- this is the comment.
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;"ka');`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE \"tech_Book\"(id int, name varchar(255));",
						FirstStatementLine:   2,
						FirstStatementColumn: 6,
						LastLine:             2,
						LastColumn:           77,
					},
					{
						Text:                 "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO \"tech_Book\" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;\"ka');",
						BaseLine:             2,
						FirstStatementLine:   4,
						FirstStatementColumn: 6,
						LastLine:             4,
						LastColumn:           81,
					},
				},
			},
		},
		{
			statement: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdflkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;
						CREATE TABLE t(a int);`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdflkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;`,
						LastLine:   7,
						LastColumn: 8,
					},
					{
						Text:                 "\n\t\t\t\t\t\tCREATE TABLE t(a int);",
						BaseLine:             7,
						FirstStatementLine:   8,
						FirstStatementColumn: 6,
						LastLine:             8,
						LastColumn:           27,
					},
				},
			},
		},
		{
			statement: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdflkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;
						CREATE TABLE t(a int);`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdflkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;`,
						LastLine:   7,
						LastColumn: 16,
					},
					{
						Text:                 "\n\t\t\t\t\t\tCREATE TABLE t(a int);",
						BaseLine:             7,
						FirstStatementLine:   8,
						FirstStatementColumn: 6,
						LastLine:             8,
						LastColumn:           27,
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
						Text:               "\r\nCREATE TABLE t1(b int);",
						BaseLine:           1,
						FirstStatementLine: 2,
						LastLine:           2,
						LastColumn:         22,
					},
				},
			},
		},
		{
			statement: `INSERT INTO "public"."table"("id","content")
			VALUES
			(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),
			(133,'knuandfan public table id'';create table t(a int, b int);set @text=''\\\\kdaminxkljasdfiebkla.unkonwn''+''abcdef.xyz\\''; local xxxyy.abcddd.mysql @text;------- '),
			(1444,'table t xyz abc a''a\\\\\\\\''b"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\');`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       "INSERT INTO \"public\".\"table\"(\"id\",\"content\")\n\t\t\tVALUES\n\t\t\t(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),\n\t\t\t(133,'knuandfan public table id'';create table t(a int, b int);set @text=''\\\\\\\\kdaminxkljasdfiebkla.unkonwn''+''abcdef.xyz\\\\''; local xxxyy.abcddd.mysql @text;------- '),\n\t\t\t(1444,'table t xyz abc a''a\\\\\\\\\\\\\\\\''b\"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\\\');",
						LastLine:   4,
						LastColumn: 91,
					},
				},
			},
		},
		{
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "INSERT INTO t VALUES ('klajfas)",
						LastLine:             0,
						LastColumn:           22,
						FirstStatementLine:   0,
						FirstStatementColumn: 0,
						Empty:                false,
					},
				},
			},
		},
		{
			statement: `INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       "INSERT INTO \"t VALUES ('klajfas)",
						LastLine:   0,
						LastColumn: 12,
					},
				},
			},
		},
		{
			statement: `/*INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:                 "/*INSERT INTO \"t VALUES ('klajfas)",
						BaseLine:             0,
						FirstStatementLine:   0,
						FirstStatementColumn: 0,
						LastLine:             0,
						LastColumn:           0,
						Empty:                false,
					},
				},
			},
		},
		{
			statement: `$$INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.SingleSQL{
					{
						Text:       "$$INSERT INTO \"t VALUES ('klajfas)",
						LastColumn: 2,
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

		res, err = SplitMultiSQLStream(strings.NewReader(test.statement), nil)
		errStr = ""
		if err != nil {
			errStr = err.Error()
		}
		require.Equal(t, test.want, resData{res, errStr}, test.statement)
	}
}
