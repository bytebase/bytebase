package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/parser"
)

type testDeparseData struct {
	stmt string
	want string
}

func runDeparseTest(t *testing.T, tests []testDeparseData) {
	p := &PostgreSQLParser{}

	for _, test := range tests {
		nodeList, err := p.Parse(parser.ParseContext{}, test.stmt)
		require.NoError(t, err)
		require.Len(t, nodeList, 1)
		res, err := p.Deparse(parser.DeparseContext{}, nodeList[0])
		require.NoError(t, err)
		require.Equal(t, test.want, res, test.stmt)
	}
}

func TestCreateTable(t *testing.T) {
	tests := []testDeparseData{
		{
			stmt: `CREATE TABLE tech_book(
				a smallint,
				b integer,
				c bigint,
				d decimal(10, 2),
				e numeric(4),
				f real,
				g double precision,
				h smallserial,
				i serial,
				j bigserial,
				k int8,
				l serial8,
				m float8,
				n int,
				o int4,
				p float4,
				q int2,
				r serial2,
				s serial4,
				t decimal)`,
			want: `CREATE TABLE "tech_book" (
    "a" smallint,
    "b" integer,
    "c" bigint,
    "d" numeric(10, 2),
    "e" numeric(4),
    "f" real,
    "g" double precision,
    "h" smallserial,
    "i" serial,
    "j" bigserial,
    "k" bigint,
    "l" bigserial,
    "m" double precision,
    "n" integer,
    "o" integer,
    "p" real,
    "q" smallint,
    "r" smallserial,
    "s" serial,
    "t" numeric
)`,
		},
		{
			stmt: `create table "TechBook"(a "user defined data type")`,
			want: `CREATE TABLE "TechBook" (
    "a" "user defined data type"
)`,
		},
		{
			stmt: `
				CREATE TABLE tech_book(
					a char(20),
					b character(30),
					c varchar(330),
					d character varying(400),
					e text
				)
			`,
			want: `CREATE TABLE "tech_book" (
    "a" character(20),
    "b" character(30),
    "c" character varying(330),
    "d" character varying(400),
    "e" text
)`,
		},
	}

	runDeparseTest(t, tests)
}
