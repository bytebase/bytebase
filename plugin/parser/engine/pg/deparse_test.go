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
			want: `CREATE TABLE "tech_book"(` +
				`"a" INT2, ` +
				`"b" INT4, ` +
				`"c" INT8, ` +
				`"d" DECIMAL(10, 2), ` +
				`"e" DECIMAL(4), ` +
				`"f" FLOAT4, ` +
				`"g" FLOAT8, ` +
				`"h" INT2 NOT NULL, ` +
				`"i" INT4 NOT NULL, ` +
				`"j" INT8 NOT NULL, ` +
				`"k" INT8, ` +
				`"l" INT8 NOT NULL, ` +
				`"m" FLOAT8, ` +
				`"n" INT4, ` +
				`"o" INT4, ` +
				`"p" FLOAT4, ` +
				`"q" INT2, ` +
				`"r" INT2 NOT NULL, ` +
				`"s" INT4 NOT NULL, ` +
				`"t" DECIMAL)`,
		},
		{
			stmt: `create table "TechBook"(a "user defined data type")`,
			want: `CREATE TABLE "TechBook"("a" "user defined data type")`,
		},
	}

	runDeparseTest(t, tests)
}
