package tsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type completionCandidateSpec struct {
	text string
	typ  base.CandidateType
}

func TestCompletionCoverageMatrix(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    []completionCandidateSpec
		notWant []completionCandidateSpec
	}{
		{
			name: "select list uses table columns",
			sql:  "SELECT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "select list uses schema qualified table columns",
			sql:  "SELECT | FROM dbo.Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "select list uses implicit table alias columns",
			sql:  "SELECT e.| FROM Employees e",
			want: columns("Id", "Name"),
		},
		{
			name: "select list uses explicit table alias columns",
			sql:  "SELECT e.| FROM Employees AS e",
			want: columns("Id", "Name"),
		},
		{
			name: "select list prefix keeps matching column available",
			sql:  "SELECT Na| FROM Employees",
			want: columns("Name"),
		},
		{
			name: "qualified column prefix keeps matching column available",
			sql:  "SELECT e.Na| FROM Employees AS e",
			want: columns("Name"),
		},
		{
			name: "cross database qualified column reference",
			sql:  "SELECT School.dbo.Student.|",
			want: columns("Id", "ParentName"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "where clause uses current table columns",
			sql:  "SELECT * FROM Employees WHERE |",
			want: columns("Id", "Name"),
		},
		{
			name: "where clause excludes select alias",
			sql:  "SELECT Id AS IdAlias FROM Employees WHERE |",
			want: columns("Id", "Name"),
			notWant: columns(
				"IdAlias",
			),
		},
		{
			name: "where clause uses qualified alias columns",
			sql:  "SELECT * FROM Employees e WHERE e.|",
			want: columns("Id", "Name"),
			notWant: columns(
				"EmployeeId",
				"Street",
			),
		},
		{
			name: "group by uses current table columns",
			sql:  "SELECT * FROM Employees GROUP BY |",
			want: columns("Id", "Name"),
		},
		{
			name: "having clause uses current table columns",
			sql:  "SELECT Name, COUNT(*) FROM Employees GROUP BY Name HAVING |",
			want: columns("Id", "Name"),
		},
		{
			name: "between expression uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id BETWEEN | AND 10",
			want: columns("Id", "Name"),
		},
		{
			name: "like expression uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Name LIKE |",
			want: columns("Id", "Name"),
		},
		{
			name: "in expression uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id IN (|)",
			want: columns("Id", "Name"),
		},
		{
			name: "in expression after comma uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id IN (1, |)",
			want: columns("Id", "Name"),
		},
		{
			name: "or predicate uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id = 1 OR |",
			want: columns("Id", "Name"),
		},
		{
			name: "and predicate after is null uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id IS NULL AND |",
			want: columns("Id", "Name"),
		},
		{
			name: "not predicate uses current table columns",
			sql:  "SELECT * FROM Employees WHERE NOT |",
			want: columns("Id", "Name"),
		},
		{
			name: "comparison expression uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id > |",
			want: columns("Id", "Name"),
		},
		{
			name: "qualified comparison expression uses alias columns",
			sql:  "SELECT * FROM Employees e WHERE e.Id = |",
			want: columns("Id", "Name"),
		},
		{
			name: "arithmetic expression uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id + | > 0",
			want: columns("Id", "Name"),
		},
		{
			name: "between upper bound uses current table columns",
			sql:  "SELECT * FROM Employees WHERE Id BETWEEN 1 AND |",
			want: columns("Id", "Name"),
		},
		{
			name: "parenthesized predicate uses current table columns",
			sql:  "SELECT * FROM Employees WHERE (|)",
			want: columns("Id", "Name"),
		},
		{
			name: "case predicate uses current table columns",
			sql:  "SELECT CASE WHEN | THEN 1 ELSE 0 END FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "case then expression uses current table columns",
			sql:  "SELECT CASE WHEN Id > 0 THEN | ELSE 0 END FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "case else expression uses current table columns",
			sql:  "SELECT CASE WHEN Id > 0 THEN 1 ELSE | END FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "function argument uses current table columns",
			sql:  "SELECT COUNT(|) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "scalar function argument uses current table columns",
			sql:  "SELECT ABS(|) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "nested function argument uses current table columns",
			sql:  "SELECT ABS(ROUND(|, 0)) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "isnull argument uses current table columns",
			sql:  "SELECT ISNULL(|, 0) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "coalesce argument uses current table columns",
			sql:  "SELECT COALESCE(NULL, |) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "nullif argument uses current table columns",
			sql:  "SELECT NULLIF(|, 0) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "cast argument uses current table columns",
			sql:  "SELECT CAST(| AS INT) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "convert argument uses current table columns",
			sql:  "SELECT CONVERT(INT, |) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "iif predicate uses current table columns",
			sql:  "SELECT IIF(|, 1, 0) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "order by includes select alias",
			sql:  "SELECT Id AS IdAlias, Name FROM Employees ORDER BY |",
			want: append(columns("Id", "Name"), column("IdAlias")),
		},
		{
			name: "order by includes tsql equals alias",
			sql:  "SELECT IdAlias = Id, Name FROM Employees ORDER BY |",
			want: append(columns("Id", "Name"), column("IdAlias")),
		},
		{
			name: "order by alias prefix keeps select alias available",
			sql:  "SELECT Id AS IdAlias, Name FROM Employees ORDER BY IdA|",
			want: columns("IdAlias"),
		},
		{
			name: "order by column prefix keeps matching column available",
			sql:  "SELECT * FROM Employees ORDER BY Na|",
			want: columns("Name"),
		},
		{
			name: "top select list uses columns",
			sql:  "SELECT TOP 10 | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "distinct select list uses columns",
			sql:  "SELECT DISTINCT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "window partition by uses columns",
			sql:  "SELECT ROW_NUMBER() OVER (PARTITION BY |) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "window partition by uses qualified alias columns",
			sql:  "SELECT ROW_NUMBER() OVER (PARTITION BY e.| ORDER BY e.Id) FROM Employees e",
			want: columns("Id", "Name"),
		},
		{
			name: "window order by uses columns",
			sql:  "SELECT ROW_NUMBER() OVER (ORDER BY |) FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "window order by uses qualified alias columns",
			sql:  "SELECT ROW_NUMBER() OVER (PARTITION BY e.Id ORDER BY e.|) FROM Employees e",
			want: columns("Id", "Name"),
		},
		{
			name: "window order by excludes select alias",
			sql:  "SELECT Id AS IdAlias, ROW_NUMBER() OVER (ORDER BY |) FROM Employees",
			want: columns("Id", "Name"),
			notWant: columns(
				"IdAlias",
			),
		},
		{
			name: "bracket quoted alias columns",
			sql:  "SELECT [e].| FROM Employees AS [e]",
			want: columns("Id", "Name"),
			notWant: columns(
				"EmployeeId",
				"Street",
			),
		},
		{
			name: "double quoted alias columns",
			sql:  `SELECT "e".| FROM Employees AS "e"`,
			want: columns("Id", "Name"),
		},
		{
			name: "order by alias columns before offset",
			sql:  "SELECT * FROM Employees e ORDER BY e.| OFFSET 10 ROWS",
			want: columns("Id", "Name"),
		},
		{
			name: "offset fetch order by keeps alias columns",
			sql:  "SELECT * FROM Employees e ORDER BY e.| OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY",
			want: columns("Id", "Name"),
		},
		{
			name: "join table reference",
			sql:  "SELECT * FROM Employees JOIN |",
			want: tables("Address", "Employees"),
		},
		{
			name: "left join table reference",
			sql:  "SELECT * FROM Employees LEFT JOIN |",
			want: tables("Address", "Employees"),
		},
		{
			name: "right join table reference",
			sql:  "SELECT * FROM Employees RIGHT JOIN |",
			want: tables("Address", "Employees"),
		},
		{
			name: "full outer join table reference",
			sql:  "SELECT * FROM Employees FULL OUTER JOIN |",
			want: tables("Address", "Employees"),
		},
		{
			name: "cross join table reference",
			sql:  "SELECT * FROM Employees CROSS JOIN |",
			want: tables("Address", "Employees"),
		},
		{
			name: "comma join table reference",
			sql:  "SELECT * FROM Employees, |",
			want: tables("Address", "Employees"),
		},
		{
			name: "cross apply table reference",
			sql:  "SELECT * FROM Employees CROSS APPLY |",
			want: tables("Address", "Employees"),
		},
		{
			name: "outer apply table reference",
			sql:  "SELECT * FROM Employees OUTER APPLY |",
			want: tables("Address", "Employees"),
		},
		{
			name: "join on uses left alias columns",
			sql:  "SELECT * FROM Employees e JOIN Address a ON e.|",
			want: columns("Id", "Name"),
		},
		{
			name: "join on excludes select alias",
			sql:  "SELECT e.Id AS IdAlias FROM Employees e JOIN Address a ON |",
			want: columns("Id", "Name", "EmployeeId", "Street"),
			notWant: columns(
				"IdAlias",
			),
		},
		{
			name: "join on uses right alias columns",
			sql:  "SELECT * FROM Employees e JOIN Address a ON e.Id = a.|",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "multi join where sees all joined table columns",
			sql:  "SELECT * FROM Employees e JOIN Address a ON e.Id = a.EmployeeId JOIN MySchema.SalaryLevel s ON e.Id = s.Id WHERE |",
			want: columns("Id", "Name", "EmployeeId", "Street", "SalaryUpBound"),
		},
		{
			name: "cross join where uses right alias columns",
			sql:  "SELECT * FROM Employees e CROSS JOIN Address a WHERE a.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "schema qualified join on uses right alias columns",
			sql:  "SELECT * FROM Employees e LEFT JOIN MySchema.SalaryLevel s ON s.|",
			want: columns("Id", "SalaryUpBound"),
		},
		{
			name: "joined order by uses right alias columns",
			sql:  "SELECT * FROM Employees e JOIN Address a ON e.Id = a.EmployeeId ORDER BY a.|",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "joined group by uses right alias columns",
			sql:  "SELECT a.EmployeeId, COUNT(*) FROM Employees e JOIN Address a ON e.Id = a.EmployeeId GROUP BY a.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "default schema table reference",
			sql:  "SELECT * FROM |",
			want: append(append(tables("Address", "Employees"), schemas("dbo", "MySchema")...), sequences("EmployeeIdSeq", "OrderSeq")...),
		},
		{
			name:    "database qualified schema reference",
			sql:     "SELECT * FROM Company.|",
			want:    schemas("dbo", "MySchema"),
			notWant: append(append(databases("Company", "School"), tables("Address", "Employees")...), sequences("EmployeeIdSeq", "OrderSeq")...),
		},
		{
			name: "database default schema table reference",
			sql:  "SELECT * FROM School..|",
			want: tables("Student"),
			notWant: tables(
				"Employees",
			),
		},
		{
			name: "linked server qualified table reference does not use local database",
			sql:  "SELECT * FROM Linked.Company.dbo.|",
			notWant: tables(
				"Address",
				"Employees",
			),
		},
		{
			name: "dbo table reference",
			sql:  "SELECT * FROM dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "alternate schema table reference",
			sql:  "SELECT * FROM MySchema.|",
			want: tables("SalaryLevel"),
			notWant: tables(
				"Employees",
			),
		},
		{
			name: "alternate schema view reference",
			sql:  "SELECT * FROM MySchema.|",
			want: views("SalaryView"),
		},
		{
			name: "cross database table reference",
			sql:  "SELECT * FROM School.dbo.|",
			want: tables("Student"),
			notWant: tables(
				"Employees",
			),
		},
		{
			name: "database qualified dbo table reference",
			sql:  "SELECT * FROM Company.dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "database qualified alternate schema table reference",
			sql:  "SELECT * FROM Company.MySchema.|",
			want: tables("SalaryLevel"),
		},
		{
			name: "bracket quoted schema table reference",
			sql:  "SELECT * FROM [dbo].|",
			want: tables("Address", "Employees"),
		},
		{
			name: "bracket quoted table alias columns",
			sql:  "SELECT * FROM [dbo].[Employees] AS [emp] WHERE [emp].|",
			want: columns("Id", "Name"),
		},
		{
			name: "schema qualified table prefix keeps matching table available",
			sql:  "SELECT * FROM dbo.Emp|",
			want: tables("Employees"),
		},
		{
			name: "cross database table prefix keeps matching table available",
			sql:  "SELECT * FROM School.dbo.St|",
			want: tables("Student"),
		},
		{
			name: "table prefix keeps matching table available",
			sql:  "SELECT * FROM Emp|",
			want: tables("Employees"),
		},
		{
			name: "schema prefix keeps matching schema available",
			sql:  "SELECT * FROM My|",
			want: schemas("MySchema"),
		},
		{
			name: "database qualified schema prefix keeps matching schema available",
			sql:  "SELECT * FROM Company.My|",
			want: schemas("MySchema"),
		},
		{
			name: "cte explicit columns are available",
			sql:  "WITH cte(EmpId, EmpName) AS (SELECT Id, Name FROM Employees) SELECT cte.| FROM cte",
			want: columns("EmpId", "EmpName"),
		},
		{
			name: "cte table is available in from",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) SELECT * FROM |",
			want: tables("cte"),
		},
		{
			name: "cte table is excluded from schema qualified from",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) SELECT * FROM dbo.|",
			want: tables("Address", "Employees"),
			notWant: tables(
				"cte",
			),
		},
		{
			name: "cte projected columns are available",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) SELECT | FROM cte",
			want: columns("Id", "Name"),
		},
		{
			name: "cte qualified projected columns are available",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) SELECT cte.| FROM cte",
			want: columns("Id", "Name"),
		},
		{
			name: "cte where uses projected columns",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) SELECT * FROM cte WHERE |",
			want: columns("Id", "Name"),
		},
		{
			name: "cte insert select uses projected columns",
			sql:  "WITH cte AS (SELECT Id, Name FROM Employees) INSERT INTO Employees SELECT | FROM cte",
			want: columns("Id", "Name"),
		},
		{
			name: "chained cte table is available",
			sql:  "WITH a AS (SELECT Id FROM Employees), b AS (SELECT Id FROM a) SELECT * FROM |",
			want: tables("a", "b"),
		},
		{
			name: "subquery select list uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE EXISTS (SELECT | FROM Address)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "subquery where uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE EXISTS (SELECT 1 FROM Address WHERE |)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "in subquery select list uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE Id IN (SELECT | FROM Address)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "apply subquery select list uses inner table columns",
			sql:  "SELECT * FROM Employees e CROSS APPLY (SELECT | FROM Address) x",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "scalar subquery select list uses inner table columns",
			sql:  "SELECT (SELECT | FROM Employees) FROM Address",
			want: columns("Id", "Name"),
		},
		{
			name: "aggregate scalar subquery uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE Id = (SELECT MAX(|) FROM Address)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "derived table alias columns",
			sql:  "SELECT d.| FROM (SELECT Id, Name FROM Employees) d",
			want: columns("Id", "Name"),
		},
		{
			name: "nested derived table alias columns",
			sql:  "SELECT x.| FROM (SELECT d.Id FROM (SELECT Id FROM Employees) d) x",
			want: columns("Id"),
		},
		{
			name: "values derived table alias columns",
			sql:  "SELECT v.| FROM (VALUES (1, 'a')) AS v(Id, ValueLabel)",
			want: columns("Id", "ValueLabel"),
		},
		{
			name: "correlated subquery can complete outer alias",
			sql:  "SELECT * FROM Employees e WHERE EXISTS (SELECT 1 FROM Address a WHERE a.EmployeeId = e.|)",
			want: columns("Id", "Name"),
		},
		{
			name: "correlated subquery can complete inner alias",
			sql:  "SELECT * FROM Employees e WHERE EXISTS (SELECT 1 FROM Address a WHERE a.| = e.Id)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "inner alias shadows outer alias",
			sql:  "SELECT * FROM Employees o WHERE EXISTS (SELECT 1 FROM Address o WHERE o.| = 1)",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "any subquery select list uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE Id = ANY (SELECT | FROM Address)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "all subquery select list uses inner table columns",
			sql:  "SELECT * FROM Employees WHERE Id > ALL (SELECT | FROM Address)",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "union right arm uses right table columns",
			sql:  "SELECT Id FROM Employees UNION SELECT | FROM Address",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "union all right arm uses right table columns",
			sql:  "SELECT Id FROM Employees UNION ALL SELECT | FROM Address",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "chained union final arm uses final table columns",
			sql:  "SELECT Id FROM Employees UNION SELECT EmployeeId FROM Address UNION SELECT | FROM MySchema.SalaryLevel",
			want: columns("Id", "SalaryUpBound"),
			notWant: columns(
				"EmployeeId",
				"Street",
			),
		},
		{
			name: "except right arm uses right table columns",
			sql:  "SELECT Id FROM Employees EXCEPT SELECT | FROM Address",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "intersect right arm uses right table columns",
			sql:  "SELECT Id FROM Employees INTERSECT SELECT | FROM Address",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "insert target table reference",
			sql:  "INSERT INTO |",
			want: tables("Address", "Employees"),
		},
		{
			name: "insert target schema table reference",
			sql:  "INSERT INTO dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "insert target cross database table reference",
			sql:  "INSERT INTO School.dbo.|",
			want: tables("Student"),
			notWant: tables(
				"Employees",
			),
		},
		{
			name: "insert target alternate schema table reference",
			sql:  "INSERT INTO MySchema.|",
			want: tables("SalaryLevel"),
		},
		{
			name: "insert column list uses target columns",
			sql:  "INSERT INTO Employees(|) VALUES (1)",
			want: columns("Id", "Name"),
		},
		{
			name: "insert schema qualified column list uses target columns",
			sql:  "INSERT INTO dbo.Employees(|) VALUES (1)",
			want: columns("Id", "Name"),
		},
		{
			name: "insert cross database column list uses target columns",
			sql:  "INSERT INTO School.dbo.Student(|) VALUES (1)",
			want: columns("Id", "ParentName"),
		},
		{
			name: "insert column list after comma uses remaining target columns",
			sql:  "INSERT INTO Employees(Id, |) VALUES (1, 'a')",
			want: columns("Name"),
			notWant: columns(
				"EmployeeId",
				"Id",
				"SalaryUpBound",
				"Street",
			),
		},
		{
			name: "insert select uses source columns",
			sql:  "INSERT INTO Employees SELECT | FROM Address",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "insert output inserted columns",
			sql:  "INSERT INTO Employees(Name) OUTPUT INSERTED.| VALUES ('a')",
			want: columns("Id", "Name"),
		},
		{
			name: "update target table reference",
			sql:  "UPDATE | SET Name = 'a'",
			want: tables("Address", "Employees"),
		},
		{
			name: "update target schema table reference",
			sql:  "UPDATE dbo.| SET Name = 'a'",
			want: tables("Address", "Employees"),
		},
		{
			name: "update target cross database table reference",
			sql:  "UPDATE School.dbo.| SET ParentName = 'a'",
			want: tables("Student"),
		},
		{
			name: "update set column uses target columns",
			sql:  "UPDATE Employees SET |",
			want: columns("Id", "Name"),
		},
		{
			name: "update cross database set column uses target columns",
			sql:  "UPDATE School.dbo.Student SET |",
			want: columns("Id", "ParentName"),
		},
		{
			name: "update where uses target columns",
			sql:  "UPDATE Employees SET Name = 'a' WHERE |",
			want: columns("Id", "Name"),
		},
		{
			name: "update set value uses target columns",
			sql:  "UPDATE Employees SET Name = |",
			want: columns("Id", "Name"),
		},
		{
			name: "update top target table reference",
			sql:  "UPDATE TOP (10) | SET Name = 'a'",
			want: tables("Address", "Employees"),
		},
		{
			name: "update alias set column uses target columns",
			sql:  "UPDATE e SET | FROM Employees e",
			want: columns("Id", "Name"),
		},
		{
			name: "update alias set value uses target columns",
			sql:  "UPDATE e SET Name = | FROM Employees e",
			want: columns("Id", "Name"),
		},
		{
			name: "update joined set value uses joined alias columns",
			sql:  "UPDATE e SET Name = a.| FROM Employees e JOIN Address a ON e.Id = a.EmployeeId",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "update from joined alias columns",
			sql:  "UPDATE e SET Name = 'x' FROM Employees e JOIN Address a ON e.Id = a.EmployeeId WHERE a.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "update output inserted columns",
			sql:  "UPDATE Employees SET Name = 'x' OUTPUT INSERTED.|",
			want: columns("Id", "Name"),
		},
		{
			name: "update output deleted columns",
			sql:  "UPDATE Employees SET Name = 'x' OUTPUT DELETED.|",
			want: columns("Id", "Name"),
		},
		{
			name: "delete target table reference",
			sql:  "DELETE FROM |",
			want: tables("Address", "Employees"),
		},
		{
			name: "delete target schema table reference",
			sql:  "DELETE FROM dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "delete target cross database table reference",
			sql:  "DELETE FROM School.dbo.|",
			want: tables("Student"),
		},
		{
			name: "delete top target table reference",
			sql:  "DELETE TOP (10) FROM |",
			want: tables("Address", "Employees"),
		},
		{
			name: "delete where uses target columns",
			sql:  "DELETE FROM Employees WHERE |",
			want: columns("Id", "Name"),
		},
		{
			name: "delete cross database where uses target columns",
			sql:  "DELETE FROM School.dbo.Student WHERE |",
			want: columns("Id", "ParentName"),
		},
		{
			name: "delete alias where uses target columns",
			sql:  "DELETE e FROM Employees e WHERE e.|",
			want: columns("Id", "Name"),
		},
		{
			name: "delete joined alias where uses joined columns",
			sql:  "DELETE e FROM Employees e JOIN Address a ON e.Id = a.EmployeeId WHERE a.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "delete output deleted columns",
			sql:  "DELETE FROM Employees OUTPUT DELETED.|",
			want: columns("Id", "Name"),
		},
		{
			name: "create index target table reference",
			sql:  "CREATE INDEX ix ON |",
			want: tables("Address", "Employees"),
		},
		{
			name: "create index target schema table reference",
			sql:  "CREATE INDEX ix ON dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "create index column list uses target columns",
			sql:  "CREATE INDEX ix ON Employees(|)",
			want: columns("Id", "Name"),
			notWant: columns(
				"EmployeeId",
				"SalaryUpBound",
				"Street",
			),
		},
		{
			name: "create index schema qualified column list uses target columns",
			sql:  "CREATE INDEX ix ON dbo.Employees(|)",
			want: columns("Id", "Name"),
		},
		{
			name: "create unique clustered index column list uses target columns",
			sql:  "CREATE UNIQUE CLUSTERED INDEX ix ON Employees(|)",
			want: columns("Id", "Name"),
		},
		{
			name: "create index include column list uses target columns",
			sql:  "CREATE INDEX ix ON Employees(Id) INCLUDE (|)",
			want: columns("Id", "Name"),
		},
		{
			name: "create view body select uses source columns",
			sql:  "CREATE VIEW v AS SELECT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "create view with schema body select uses source columns",
			sql:  "CREATE VIEW MySchema.v AS SELECT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "create procedure body select uses source columns",
			sql:  "CREATE PROCEDURE p AS SELECT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "create procedure parameter type name",
			sql:  "CREATE PROCEDURE p @Name | AS SELECT 1",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "create table references table",
			sql:  "CREATE TABLE NewTable (EmployeeId INT REFERENCES |)",
			want: tables("Address", "Employees"),
		},
		{
			name: "create table references cross database table",
			sql:  "CREATE TABLE NewTable (StudentId INT REFERENCES School.dbo.|)",
			want: tables("Student"),
			notWant: tables(
				"Employees",
			),
		},
		{
			name: "create table foreign key source column list",
			sql:  "CREATE TABLE NewTable (EmployeeId INT, FOREIGN KEY (|) REFERENCES Employees(Id))",
			want: columns("EmployeeId"),
			notWant: columns(
				"Id",
				"Name",
				"Street",
			),
		},
		{
			name: "create table named foreign key source column list",
			sql:  "CREATE TABLE NewTable (EmployeeId INT, CONSTRAINT fk FOREIGN KEY (|) REFERENCES Employees(Id))",
			want: columns("EmployeeId"),
		},
		{
			name: "create table references column list",
			sql:  "CREATE TABLE NewTable (EmployeeId INT REFERENCES Employees(|))",
			want: columns("Id", "Name"),
			notWant: columns(
				"EmployeeId",
				"SalaryUpBound",
				"Street",
			),
		},
		{
			name: "create table schema qualified references column list",
			sql:  "CREATE TABLE NewTable (EmployeeId INT REFERENCES dbo.Employees(|))",
			want: columns("Id", "Name"),
		},
		{
			name: "create table cross database references column list",
			sql:  "CREATE TABLE NewTable (StudentId INT REFERENCES School.dbo.Student(|))",
			want: columns("Id", "ParentName"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "create table type name",
			sql:  "CREATE TABLE NewTable (Name |)",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "create table type name after comma",
			sql:  "CREATE TABLE NewTable (Id INT, Name |)",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "declare variable type name",
			sql:  "DECLARE @Name |",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "alter table target reference",
			sql:  "ALTER TABLE |",
			want: tables("Address", "Employees"),
		},
		{
			name: "alter table target schema reference",
			sql:  "ALTER TABLE dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "alter table cross database target reference",
			sql:  "ALTER TABLE School.dbo.|",
			want: tables("Student"),
		},
		{
			name: "alter table add column type name",
			sql:  "ALTER TABLE Employees ADD Name |",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "alter table references table",
			sql:  "ALTER TABLE Employees ADD CONSTRAINT fk FOREIGN KEY (Id) REFERENCES |",
			want: tables("Address", "Employees"),
		},
		{
			name: "alter table references column list",
			sql:  "ALTER TABLE Employees ADD CONSTRAINT fk FOREIGN KEY (Id) REFERENCES Address(|)",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "drop table reference",
			sql:  "DROP TABLE |",
			want: tables("Address", "Employees"),
		},
		{
			name: "drop table prefix keeps matching table available",
			sql:  "DROP TABLE Emp|",
			want: tables("Employees"),
		},
		{
			name: "drop table if exists reference",
			sql:  "DROP TABLE IF EXISTS |",
			want: tables("Address", "Employees"),
		},
		{
			name: "drop table if exists database qualified schema reference",
			sql:  "DROP TABLE IF EXISTS School.|",
			want: schemas("dbo"),
			notWant: tables(
				"Address",
				"Employees",
			),
		},
		{
			name: "drop schema qualified table reference",
			sql:  "DROP TABLE dbo.|",
			want: tables("Address", "Employees"),
		},
		{
			name: "drop cross database table reference",
			sql:  "DROP TABLE School.dbo.|",
			want: tables("Student"),
		},
		{
			name: "drop view schema reference",
			sql:  "DROP VIEW MySchema.|",
			want: views("SalaryView"),
		},
		{
			name: "drop procedure schema reference",
			sql:  "DROP PROCEDURE dbo.|",
			want: routines("SyncEmployees"),
		},
		{
			name: "drop procedure database qualified schema reference",
			sql:  "DROP PROCEDURE Company.|",
			want: schemas("dbo", "MySchema"),
			notWant: routines(
				"SyncEmployees",
				"SyncSalary",
			),
		},
		{
			name: "drop procedure if exists database qualified schema reference",
			sql:  "DROP PROCEDURE IF EXISTS Company.|",
			want: schemas("dbo", "MySchema"),
			notWant: routines(
				"SyncEmployees",
				"SyncSalary",
			),
		},
		{
			name: "drop procedure prefix keeps matching routine available",
			sql:  "DROP PROCEDURE Sync|",
			want: routines("SyncEmployees"),
		},
		{
			name: "drop sequence schema reference",
			sql:  "DROP SEQUENCE dbo.|",
			want: sequences("EmployeeIdSeq", "OrderSeq"),
		},
		{
			name: "drop sequence prefix keeps matching sequence available",
			sql:  "DROP SEQUENCE Employee|",
			want: sequences("EmployeeIdSeq"),
		},
		{
			name: "drop database reference",
			sql:  "DROP DATABASE |",
			want: databases("Company", "School"),
		},
		{
			name: "drop database prefix keeps matching database available",
			sql:  "DROP DATABASE Comp|",
			want: databases("Company"),
		},
		{
			name: "use database reference",
			sql:  "USE |",
			want: databases("Company", "School"),
		},
		{
			name: "use database prefix keeps matching database available",
			sql:  "USE Sch|",
			want: databases("School"),
		},
		{
			name: "next value for sequence reference",
			sql:  "SELECT NEXT VALUE FOR |",
			want: sequences("EmployeeIdSeq", "OrderSeq"),
		},
		{
			name: "next value for sequence prefix",
			sql:  "SELECT NEXT VALUE FOR Employee|",
			want: sequences("EmployeeIdSeq"),
			notWant: sequences(
				"OrderSeq",
			),
		},
		{
			name: "schema qualified next value for sequence reference",
			sql:  "SELECT NEXT VALUE FOR dbo.|",
			want: sequences("EmployeeIdSeq", "OrderSeq"),
		},
		{
			name: "alternate schema next value for sequence reference",
			sql:  "SELECT NEXT VALUE FOR MySchema.|",
			want: sequences("SalarySeq"),
			notWant: sequences(
				"EmployeeIdSeq",
			),
		},
		{
			name: "cross database next value for sequence reference",
			sql:  "SELECT NEXT VALUE FOR School.dbo.|",
			want: sequences("StudentSeq"),
			notWant: sequences(
				"EmployeeIdSeq",
			),
		},
		{
			name: "database qualified next value for schema reference",
			sql:  "SELECT NEXT VALUE FOR Company.|",
			want: schemas("dbo", "MySchema"),
			notWant: sequences(
				"EmployeeIdSeq",
				"OrderSeq",
			),
		},
		{
			name: "truncate table reference",
			sql:  "TRUNCATE TABLE |",
			want: tables("Address", "Employees"),
		},
		{
			name: "truncate cross database table reference",
			sql:  "TRUNCATE TABLE School.dbo.|",
			want: tables("Student"),
		},
		{
			name: "merge target table reference",
			sql:  "MERGE INTO |",
			want: tables("Address", "Employees"),
		},
		{
			name: "merge source table reference",
			sql:  "MERGE INTO Employees USING |",
			want: tables("Address", "Employees"),
		},
		{
			name: "merge on uses target and source columns",
			sql:  "MERGE INTO Employees USING Address ON |",
			want: columns("Id", "Name", "EmployeeId", "Street"),
		},
		{
			name: "merge on uses aliased target and source columns",
			sql:  "MERGE INTO Employees AS target USING Address AS source ON source.|",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "merge on uses cte source columns",
			sql:  "WITH source AS (SELECT EmployeeId, Street FROM Address) MERGE INTO Employees AS target USING source ON |",
			want: columns("Id", "Name", "EmployeeId", "Street"),
		},
		{
			name: "merge on uses subquery source columns",
			sql:  "MERGE INTO Employees AS target USING (SELECT EmployeeId, Street FROM Address) AS source ON source.|",
			want: columns("EmployeeId", "Street"),
			notWant: columns(
				"Name",
			),
		},
		{
			name: "merge when keyword",
			sql:  "MERGE INTO Employees USING Address ON Employees.Id = Address.EmployeeId WHEN |",
			want: keywords("MATCHED", "NOT"),
		},
		{
			name: "merge matched update set uses target columns",
			sql:  "MERGE INTO Employees USING Address ON Employees.Id = Address.EmployeeId WHEN MATCHED THEN UPDATE SET |",
			want: columns("Id", "Name"),
		},
		{
			name: "merge matched update value uses source columns",
			sql:  "MERGE INTO Employees USING Address ON Employees.Id = Address.EmployeeId WHEN MATCHED THEN UPDATE SET Name = Address.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "merge output inserted columns",
			sql:  "MERGE INTO Employees USING Address ON Employees.Id = Address.EmployeeId WHEN MATCHED THEN UPDATE SET Name = Address.Street OUTPUT INSERTED.|",
			want: columns("Id", "Name"),
		},
		{
			name: "merge output deleted columns",
			sql:  "MERGE INTO Employees USING Address ON Employees.Id = Address.EmployeeId WHEN MATCHED THEN UPDATE SET Name = Address.Street OUTPUT DELETED.|",
			want: columns("Id", "Name"),
		},
		{
			name: "table hint keyword",
			sql:  "SELECT * FROM Employees WITH (|)",
			want: keywords("NOLOCK", "UPDLOCK"),
		},
		{
			name: "table hint keyword after comma",
			sql:  "SELECT * FROM Employees WITH (NOLOCK, |)",
			want: keywords("UPDLOCK", "TABLOCK"),
		},
		{
			name: "table hint keyword after multiple hints",
			sql:  "SELECT * FROM Employees WITH (NOLOCK, ROWLOCK, |)",
			want: keywords("HOLDLOCK", "TABLOCKX"),
		},
		{
			name: "join table hint keeps joined alias columns",
			sql:  "SELECT * FROM Employees e JOIN Address a WITH (NOLOCK) ON a.|",
			want: columns("EmployeeId", "Street"),
		},
		{
			name: "for xml mode",
			sql:  "SELECT * FROM Employees FOR XML |",
			want: keywords("PATH", "RAW", "AUTO", "EXPLICIT"),
		},
		{
			name: "for json mode",
			sql:  "SELECT * FROM Employees FOR JSON |",
			want: keywords("PATH", "AUTO"),
		},
		{
			name: "openjson with column type name",
			sql:  "SELECT * FROM OPENJSON(@json) WITH (Name |)",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "openjson with second column type name",
			sql:  "SELECT * FROM OPENJSON(@json) WITH (Name NVARCHAR(100), Age |)",
			want: keywords("INT", "NVARCHAR"),
		},
		{
			name: "pivot in list uses source columns",
			sql:  "SELECT * FROM Employees PIVOT (COUNT(Id) FOR Name IN (|)) p",
			want: columns("Id", "Name"),
		},
		{
			name: "unpivot in list uses source columns",
			sql:  "SELECT * FROM Employees UNPIVOT (Value FOR Field IN (|)) u",
			want: columns("Id", "Name"),
		},
		{
			name: "execute procedure reference",
			sql:  "EXEC |",
			want: routines("SyncEmployees"),
		},
		{
			name: "execute procedure prefix",
			sql:  "EXEC Sync|",
			want: routines("SyncEmployees"),
		},
		{
			name: "execute database qualified schema reference",
			sql:  "EXEC Company.|",
			want: schemas("dbo", "MySchema"),
			notWant: routines(
				"SyncEmployees",
				"SyncSalary",
			),
		},
		{
			name: "alternate schema execute procedure reference",
			sql:  "EXEC MySchema.|",
			want: routines("SyncSalary"),
			notWant: routines(
				"SyncEmployees",
			),
		},
		{
			name: "cross database execute procedure reference",
			sql:  "EXEC School.dbo.|",
			want: routines("SyncStudents"),
			notWant: routines(
				"SyncEmployees",
			),
		},
		{
			name: "option query hint",
			sql:  "SELECT * FROM Employees OPTION (|)",
			want: keywords("RECOMPILE", "MAXDOP"),
		},
		{
			name: "option query hint after comma",
			sql:  "SELECT * FROM Employees OPTION (HASH JOIN, |)",
			want: keywords("RECOMPILE", "MAXDOP"),
		},
		{
			name: "multi statement skips earlier invalid sql",
			sql:  "SELECT FROM broken; SELECT | FROM Employees",
			want: columns("Id", "Name"),
		},
		{
			name: "multi statement update after select",
			sql:  "SELECT 1; UPDATE | SET Name = 'a'",
			want: tables("Address", "Employees"),
		},
		{
			name: "go batch separator starts new table reference",
			sql:  "SELECT 1\nGO\nSELECT * FROM |",
			want: tables("Address", "Employees"),
		},
	}

	getter, lister := buildMockDatabaseMetadataGetterLister()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement, caretLine, caretPosition := getCaretPosition(tc.sql)
			got, err := Completion(context.Background(), base.CompletionContext{
				Scene:             base.SceneTypeAll,
				DefaultDatabase:   "Company",
				Metadata:          getter,
				ListDatabaseNames: lister,
			}, statement, caretLine, caretPosition)
			require.NoError(t, err)

			for _, want := range tc.want {
				require.Truef(t, hasCompletionCandidate(got, want), "missing %s candidate %q in %#v", want.typ, want.text, got)
			}
			for _, notWant := range tc.notWant {
				require.Falsef(t, hasCompletionCandidate(got, notWant), "unexpected %s candidate %q in %#v", notWant.typ, notWant.text, got)
			}
		})
	}
}

func hasCompletionCandidate(candidates []base.Candidate, want completionCandidateSpec) bool {
	for _, candidate := range candidates {
		if candidate.Text == want.text && candidate.Type == want.typ {
			return true
		}
	}
	return false
}

func column(text string) completionCandidateSpec {
	return completionCandidateSpec{text: text, typ: base.CandidateTypeColumn}
}

func columns(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeColumn, texts...)
}

func tables(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeTable, texts...)
}

func views(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeView, texts...)
}

func schemas(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeSchema, texts...)
}

func databases(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeDatabase, texts...)
}

func sequences(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeSequence, texts...)
}

func keywords(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeKeyword, texts...)
}

func routines(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeRoutine, texts...)
}

func completionSpecs(typ base.CandidateType, texts ...string) []completionCandidateSpec {
	specs := make([]completionCandidateSpec, 0, len(texts))
	for _, text := range texts {
		specs = append(specs, completionCandidateSpec{
			text: text,
			typ:  typ,
		})
	}
	return specs
}
