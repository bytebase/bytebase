package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testData struct {
	statement string
	want      resData
}

type resData struct {
	res []SingleSQL
	err error
}

func TestPGSplitMultiSQL(t *testing.T) {
	tests := []testData{
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []SingleSQL{
					{
						Text: "CREATE TABLE t(a int);",
						Line: 1,
					},
					{
						Text: "CREATE TABLE t1(a int)",
						Line: 1,
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
						Text: `CREATE TABLE "tech_Book"(id int, name varchar(255));`,
						Line: 1,
					},
					{
						Text: `INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						Line: 2,
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
						Line: 2,
					},
					{
						Text: `-- this is the comment.
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
						Line: 4,
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
						Line: 1,
					},
					{
						Text: `CREATE TABLE t(a int);`,
						Line: 9,
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
						Line: 1,
					},
					{
						Text: `CREATE TABLE t(a int);`,
						Line: 9,
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
						Text: "CREATE TABLE t\r\n(a int);",
						Line: 1,
					},
					{
						Text: "CREATE TABLE t1(b int);",
						Line: 3,
					},
				},
			},
		},
		{
			statement: `INSERT INTO t VALUES ('klajfas)`,
			want: resData{
				err: fmt.Errorf("invalid string: not found delimiter: ', but found EOF"),
			},
		},
		{
			statement: `INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: fmt.Errorf("invalid indentifier: not found delimiter: \", but found EOF"),
			},
		},
		{
			statement: `/*INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: fmt.Errorf("invalid comment: not found */, but found EOF"),
			},
		},
		{
			statement: `$$INSERT INTO "t VALUES ('klajfas)`,
			want: resData{
				err: fmt.Errorf("scanTo failed: delimiter \"$$\" not found"),
			},
		},
	}

	for _, test := range tests {
		res, err := SplitMultiSQL(Postgres, test.statement)
		require.Equal(t, test.want, resData{res, err}, test.statement)
	}
}
