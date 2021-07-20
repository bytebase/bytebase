// pgrestore is a library for restoring Postgres database schemas and data provided by bytebase.com.
package pgrestore

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/bin/bb/connect"
)

var (
	asToken = regexp.MustCompile("(AS )[$]([a-z]+)[$]")
)

// Restore restores the schema of a Postgres instance.
func Restore(conn *connect.PostgresConnect, sc *bufio.Scanner) error {
	s := ""
	tokenName := ""
	for sc.Scan() {
		line := sc.Text()

		execute := false

		switch {
		case s == "" && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case tokenName != "":
			if strings.Contains(line, tokenName) {
				tokenName = ""
			}
		default:
			token := asToken.FindString(line)
			if token != "" {
				identifier := token[3:]
				rest := line[strings.Index(line, identifier)+len(identifier):]
				if !strings.Contains(rest, identifier) {
					tokenName = identifier
				}
			}
		}
		s = s + line + "\n"
		if strings.HasSuffix(line, ";") && tokenName == "" {
			execute = true
		}

		if execute {
			_, err := conn.DB.Exec(s)
			if err != nil {
				return fmt.Errorf("execute query %q failed: %v", s, err)
			}
			s = ""
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	return nil
}
