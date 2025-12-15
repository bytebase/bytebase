package pg

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

func TestPGSplitMultiSQL(t *testing.T) {
	bigSQL := generateOneMBInsert()
	tests := []splitTestData{
		{
			statement: `-- klsjdfjasldf
			-- klsjdflkjaskldfj
			`,
			want: resData{
				res: []base.Statement{
					{
						Text:     "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n\t\t\t",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 3, Column: 4},
						End:      &storepb.Position{Line: 3, Column: 3},
						Empty:    true,
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
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 16},
					},
					{
						Text:  "\n\t\t\t/* sdfasdf */",
						End:   &storepb.Position{Line: 2, Column: 4},
						Start: &storepb.Position{Line: 2, Column: 17},
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
				res: []base.Statement{
					{
						Text:  `select * from t;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 16},
					},
					{
						Text:  "\n\t\t\t/* sdfasdf */;",
						Start: &storepb.Position{Line: 2, Column: 17},
						End:   &storepb.Position{Line: 2, Column: 17},
						Empty: true,
					},
					{
						Text:     "\n\t\t\tselect * from t;",
						BaseLine: 1,
						Start:    &storepb.Position{Line: 3, Column: 4},
						End:      &storepb.Position{Line: 3, Column: 19},
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
					},
					{
						Text:  " CREATE TABLE t1(a int)",
						Start: &storepb.Position{Line: 1, Column: 28},
						End:   &storepb.Position{Line: 1, Column: 49},
					},
				},
			},
		},
		{
			statement: `CREATE TABLE "tech_Book"(id int, name varchar(255));
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;"ka');`,
			want: resData{
				res: []base.Statement{
					{
						Text:  `CREATE TABLE "tech_Book"(id int, name varchar(255));`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 52},
					},
					{
						Text:  "\n\t\t\t\t\t\tINSERT INTO \"tech_Book\" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;\"ka');",
						Start: &storepb.Position{Line: 2, Column: 7},
						End:   &storepb.Position{Line: 2, Column: 82},
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
				res: []base.Statement{
					{
						Text:  "\n\t\t\t\t\t\t/* this is the comment. */\n\t\t\t\t\t\tCREATE /* inline comment */TABLE \"tech_Book\"(id int, name varchar(255));",
						Start: &storepb.Position{Line: 3, Column: 7},
						End:   &storepb.Position{Line: 3, Column: 78},
					},
					{
						Text:     "\n\t\t\t\t\t\t-- this is the comment.\n\t\t\t\t\t\tINSERT INTO \"tech_Book\" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;\"ka');",
						BaseLine: 2,
						Start:    &storepb.Position{Line: 5, Column: 7},
						End:      &storepb.Position{Line: 5, Column: 82},
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
				res: []base.Statement{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdflkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 8, Column: 9},
					},
					{
						Text:     "\n\t\t\t\t\t\tCREATE TABLE t(a int);",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 9, Column: 7},
						End:      &storepb.Position{Line: 9, Column: 28},
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
				res: []base.Statement{
					{
						Text: `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdflkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;`,
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 8, Column: 17},
					},
					{
						Text:     "\n\t\t\t\t\t\tCREATE TABLE t(a int);",
						BaseLine: 7,
						Start:    &storepb.Position{Line: 9, Column: 7},
						End:      &storepb.Position{Line: 9, Column: 28},
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
						Text:  "CREATE TABLE t\r\n(a int);",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 2, Column: 8},
					},
					{
						Text:     "\r\nCREATE TABLE t1(b int);",
						BaseLine: 1,
						Start:    &storepb.Position{Line: 3, Column: 1},
						End:      &storepb.Position{Line: 3, Column: 23},
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
				res: []base.Statement{
					{
						Text:  "INSERT INTO \"public\".\"table\"(\"id\",\"content\")\n\t\t\tVALUES\n\t\t\t(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),\n\t\t\t(133,'knuandfan public table id'';create table t(a int, b int);set @text=''\\\\\\\\kdaminxkljasdfiebkla.unkonwn''+''abcdef.xyz\\\\''; local xxxyy.abcddd.mysql @text;------- '),\n\t\t\t(1444,'table t xyz abc a''a\\\\\\\\\\\\\\\\''b\"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\\\');",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 5, Column: 92},
					},
				},
			},
		},
		{
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				res: []base.Statement{
					{
						Text:  "INSERT INTO t VALUES ('klajfas)",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 23},
						Empty: false,
					},
				},
			},
		},
		{
			statement: `INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.Statement{
					{
						Text:  "INSERT INTO \"t VALUES ('klajfas)",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 13},
					},
				},
			},
		},
		{
			statement: `/*INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.Statement{
					{
						Text:     "/*INSERT INTO \"t VALUES ('klajfas)",
						BaseLine: 0,
						Start:    &storepb.Position{Line: 1, Column: 1},
						End:      &storepb.Position{Line: 1, Column: 1},
						Empty:    false,
					},
				},
			},
		},
		{
			statement: `$$INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				res: []base.Statement{
					{
						Text:  "$$INSERT INTO \"t VALUES ('klajfas)",
						Start: &storepb.Position{Line: 1, Column: 1},
						End:   &storepb.Position{Line: 1, Column: 3},
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
