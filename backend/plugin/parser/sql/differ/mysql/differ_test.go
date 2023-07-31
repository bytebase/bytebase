package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractUnsupportObjNameAndType(t *testing.T) {
	tests := []struct {
		stmt     string
		wantTp   objectType
		wantName string
	}{
		{
			stmt:     "CREATE DEFINER=`root`@`%` TRIGGER xcytestT \t\nBEFORE \n INSERT ON xcytest FOR EACH ROW\n BEGIN\n\tSET new.code=REPLACE(UUID(), '-', ''); \nEND ;;",
			wantTp:   trigger,
			wantName: "xcytestT",
		},
		{
			stmt:     "CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;",
			wantTp:   trigger,
			wantName: "ins_sum",
		},
		{
			stmt:     "create trigger `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;",
			wantTp:   trigger,
			wantName: "ins_sum",
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` PROCEDURE `citycount` (IN `country` CHAR(3), OUT `cities` INT)\n" +
				"BEGIN\n" +
				"	SELECT COUNT(*) INTO cities FROM world.city\n" +
				"WHERE CountryCode = country;\n" +
				"END//",
			wantTp:   procedure,
			wantName: "citycount",
		},
		{
			stmt: "create procedure `citycount` (IN `country` CHAR(3), OUT `cities` INT)\n" +
				"BEGIN\n" +
				"	SELECT COUNT(*) INTO cities FROM world.city\n" +
				"WHERE CountryCode = country;\n" +
				"END//",
			wantTp:   procedure,
			wantName: "citycount",
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` FUNCTION `hello` (s CHAR(20)) RETURNS CHAR(50) DETERMINISTIC\n" +
				"RETURN CONCAT('Hello, ',s,'!');",
			wantTp:   function,
			wantName: "hello",
		},
		{
			stmt: "create function `hello` (s CHAR(20)) RETURNS CHAR(50) DETERMINISTIC\n" +
				"RETURN CONCAT('Hello, ',s,'!');",
			wantTp:   function,
			wantName: "hello",
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` EVENT `test_event_01` ON SCHEDULE AT CURRENT_TIMESTAMP \n" +
				"DO\n" +
				"	INSERT INTO message(message, created_at)\n" +
				"	VALUES('test event', NOW());",
			wantTp:   event,
			wantName: "test_event_01",
		},
		{
			stmt: "create event `test_event_01` ON SCHEDULE AT CURRENT_TIMESTAMP \n" +
				"DO\n" +
				"	INSERT INTO message(message, created_at)\n" +
				"	VALUES('test event', NOW());",
			wantTp:   event,
			wantName: "test_event_01",
		},
	}

	a := require.New(t)
	for _, test := range tests {
		gotName, gotTp, err := extractUnsupportObjNameAndType(test.stmt)
		a.NoError(err)
		a.Equal(test.wantTp, gotTp)
		a.Equal(test.wantName, gotName)
	}
}

type testCase struct {
	old  string
	new  string
	want string
}

func testDiffWithoutDisableForeignKeyCheck(t *testing.T, testCases []testCase) {
	a := require.New(t)
	mysqlDiffer := &SchemaDiffer{}
	for _, test := range testCases {
		out, err := mysqlDiffer.SchemaDiff(test.old, test.new, true /* ignoreCaseSensitive */)
		a.NoError(err)
		if len(out) > 0 {
			a.Equal(disableFKCheckStmt, out[:len(disableFKCheckStmt)])
			out = out[len(disableFKCheckStmt):]
			a.Equal(enableFKCheckStmt, out[len(out)-len(enableFKCheckStmt):])
			out = out[:len(out)-len(enableFKCheckStmt)]
		}
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}
