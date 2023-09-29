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
		},
		{
			statement: `select * from t;
			/* sdfasdf */`,
			want: resData{
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
				res: []base.SingleSQL{
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
