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
	res []string
	err error
}

func TestPGSplitMultiSQL(t *testing.T) {
	tests := []testData{
		{
			statement: "    CREATE TABLE t(a int); CREATE TABLE t1(a int)",
			want: resData{
				res: []string{
					"CREATE TABLE t(a int);",
					"CREATE TABLE t1(a int)",
				},
			},
		},
		{
			statement: `CREATE TABLE "tech_Book"(id int, name varchar(255));
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
			want: resData{
				res: []string{
					`CREATE TABLE "tech_Book"(id int, name varchar(255));`,
					`INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
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
				res: []string{
					`/* this is the comment. */
						CREATE /* inline comment */TABLE "tech_Book"(id int, name varchar(255));`,
					`-- this is the comment.
						INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafa\'jdfl;"ka');`,
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
				res: []string{`CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdf\'lkasdf"asdklf\\');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$$;`,
					`CREATE TABLE t(a int);`,
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
				res: []string{`CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
						LANGUAGE SQL
						AS $tag_name$
						/*this is the comment */
						INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdf\'lkasdf"asdklf');
						-- this is the comment
						INSERT INTO tbl VALUES ('fasf_bkdjlfa');
						$tag_name$;`,
					`CREATE TABLE t(a int);`,
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
