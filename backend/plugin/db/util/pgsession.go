package util

import (
	"database/sql"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

// pgLocations caches locations by PostgreSQL TimeZone setting. time.LoadLocation
// reparses the zoneinfo file on every call for a named zone, which costs microseconds
// and kilobytes; a query result holds one location, often for every cell.
var pgLocations sync.Map

// PgSessionLocation reports the location to render timestamptz values in for the next
// statement on conn: the one named by the session's TimeZone, which the server reports
// on the connection and resends whenever a statement changes it.
//
// Read it once per statement, immediately before running that statement. A statement
// then sees any SET TimeZone that ran before it, and one result set decodes under one
// location, because PostgreSQL reports a change at command completion, after the rows.
//
// PostgreSQL also accepts POSIX specifications such as "+08:00" and
// "EST5EDT,M3.2.0/2,M11.1.0/2", which time.LoadLocation rejects. Those fall back to UTC,
// keeping the instant correct, rather than failing every timestamptz read on the
// instance.
func PgSessionLocation(conn *sql.Conn) *time.Location {
	location := time.UTC
	// A failure here leaves the zone unknown, which UTC already covers.
	_ = conn.Raw(func(driverConn any) error {
		pgxConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return nil
		}
		location = pgLocation(pgxConn.Conn().PgConn().ParameterStatus("TimeZone"))
		return nil
	})
	return location
}

func pgLocation(name string) *time.Location {
	if name == "" || name == "UTC" {
		return time.UTC
	}
	if cached, ok := pgLocations.Load(name); ok {
		location, ok := cached.(*time.Location)
		if ok {
			return location
		}
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		location = time.UTC
	}
	pgLocations.Store(name, location)
	return location
}
