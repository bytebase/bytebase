package mysql

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
	autoRandSymbol      = "AUTO_RANDOM"
)

var (
	// https://dev.mysql.com/doc/refman/8.0/en/data-type-defaults.html
	// expressionDefaultOnlyTypes is a list of types that only accept expression as default
	// value. While we restore the following types, we should not restore the default null.
	// +-------+--------------------------------------------------------------------+
	// | Table | Create Table                                                       |
	// +-------+--------------------------------------------------------------------+
	// | u     | CREATE TABLE `u` (                                                 |
	// |       |   `b` blob,                                                        |
	// |       |   `t` text,                                                        |
	// |       |   `g` geometry DEFAULT NULL,                                       |
	// |       |   `j` json DEFAULT NULL                                            |
	// |       | ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
	// +-------+--------------------------------------------------------------------+.
	expressionDefaultOnlyTypes = map[string]bool{
		// BLOB & TEXT
		// https://dev.mysql.com/doc/refman/8.0/en/blob.html
		"TINYBLOB":   true,
		"BLOB":       true,
		"MEIDUMBLOB": true,
		"LONGBLOB":   true,
		"TINYTEXT":   true,
		"TEXT":       true,
		"MEDIUMTEXT": true,
		"LONGTEXT":   true,

		// In practice, the following types restore the default null by mysqldump.
		// // GEOMETRY
		// "GEOMETRY": true,
		// // JSON
		// // https://dev.mysql.com/doc/refman/8.0/en/json.html
		// "JSON": true,
	}
)
