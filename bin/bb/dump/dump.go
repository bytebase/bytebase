// dump is a library for dumping database schemas provided by bytebase.com.
package dump

import (
	"fmt"
	"os"
)

// Dump exports the schema of a database instance.
// All non-system databases will be exported to the input directory in the format of database_name.sql for each database.
// When directory isn't specified, the schema will be exported to stdout.
func Dump(databaseType, username, password, hostname, port, directory string) error {
	if directory != "" {
		dirInfo, err := os.Stat(directory)
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", directory)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf("path %q isn't a directory", directory)
		}
	}

	switch databaseType {
	case "mysql":
		return MysqlDump(username, password, hostname, port, directory)
	default:
		return fmt.Errorf("database type %q not supported", databaseType)
	}
}
