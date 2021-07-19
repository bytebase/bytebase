// connect is a library for establishing connection to databases provided by bytebase.com.
package connect

import (
	"database/sql"
	"fmt"
	"strings"
)

// MysqlConnect is a class for MySQL database connections.
type PostgresConnect struct {
	baseDNS string
	// DB is a shared database object across actions for different databases.
	// Use switchDatabase() for connecting to a different database.
	DB *sql.DB
}

// New creates a new Postgres connection.
func NewPostgres(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (*PostgresConnect, error) {
	if (sslCert == "" && sslKey != "") || (sslCert != "" && sslKey == "") {
		return nil, fmt.Errorf("ssl-cert and ssl-key must be both set or unset.")
	}

	dns, err := guessDNS(username, password, hostname, port, database, sslCA, sslCert, sslKey)
	if err != nil {
		return nil, err
	}

	// db is closed in the dumper closer.
	db, err := sql.Open("postgres", dns)
	if err != nil {
		return nil, err
	}
	return &PostgresConnect{
		baseDNS: dns,
		DB:      db,
	}, nil
}

// Close closes the connection.
func (c *PostgresConnect) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// guessDNS will guess the dns of a valid DB connection.
func guessDNS(username, password, hostname, port, database, sslCA, sslCert, sslKey string) (string, error) {
	// dbname is guessed if not specified.
	m := map[string]string{
		"host":     hostname,
		"port":     port,
		"user":     username,
		"password": password,
	}

	if sslCA == "" {
		m["sslmode"] = "disable"
	} else {
		m["sslmode"] = "verify-ca"
		m["sslrootcert"] = sslCA
		if sslCert != "" && sslKey != "" {
			m["sslcert"] = sslCert
			m["sslkey"] = sslKey
		}
	}
	var tokens []string
	for k, v := range m {
		if v != "" {
			tokens = append(tokens, fmt.Sprintf("%s=%s", k, v))
		}
	}
	dns := strings.Join(tokens, " ")

	var guesses []string
	if database != "" {
		guesses = append(guesses, dns+" dbname="+database)
	} else {
		// Guess default database postgres, template1.
		guesses = append(guesses, dns)
		guesses = append(guesses, dns+" dbname=postgres")
		guesses = append(guesses, dns+" dbname=template1")
	}

	for _, dns := range guesses {
		db, err := sql.Open("postgres", dns)
		if err != nil {
			continue
		}
		defer db.Close()

		if err = db.Ping(); err != nil {
			continue
		}
		return dns, nil
	}
	return "", fmt.Errorf("cannot find valid dns for connection")
}

// SwitchDatabase switches to a different database.
func (c *PostgresConnect) SwitchDatabase(dbName string) error {
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			return err
		}
	}

	dns := c.baseDNS + " dbname=" + dbName
	db, err := sql.Open("postgres", dns)
	if err != nil {
		return err
	}
	c.DB = db
	return nil
}
