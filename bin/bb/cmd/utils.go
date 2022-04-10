package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

// driver://[username[:password]@]host[:port]/[database][?param1=value1&...&paramN=valueN]
var dsnPattern *regexp.Regexp

func init() {
	dsnPattern = regexp.MustCompile(
		`^(?P<driver>.+?)(?::\/\/)` + // driver://
			`(?:(?P<username>.+?)(?::(?P<password>.+?))?@)?` + // [username[:password]@]
			`(?P<host>.+?)(?::(?P<port>.+?))?` + // host[:port]
			`\/(?P<database>.+?)` + // /[database]
			`(?:\?(?P<params>[^\?]*))?$`) // [?param1=value1&...&paramN=valueN]
}

type dataSource struct {
	driver   string
	username string
	password string
	host     string
	port     string
	database string
	params   map[string]string

	sslCA   string // server-ca.pem
	sslCert string // client-cert.pem
	sslKey  string // client-key.pem
}

func parseDSN(dsn string) (*dataSource, error) {
	if !dsnPattern.MatchString(dsn) {
		return nil, fmt.Errorf("invalid dsn: %s", dsn)
	}

	ds := new(dataSource)
	ds.params = make(map[string]string)

	matches := dsnPattern.FindStringSubmatch(dsn)
	names := dsnPattern.SubexpNames()

	for i, match := range matches {
		switch names[i] {
		case "driver":
			ds.driver = match
		case "username":
			ds.username = match
		case "password":
			ds.password = match
		case "host":
			ds.host = match
		case "port":
			ds.port = match
		case "database":
			ds.database = match
		case "params":
			if len(match) == 0 {
				// no params
				continue
			}
			for _, v := range strings.Split(match, "&") {
				param := strings.SplitN(v, "=", 2)
				if len(param) != 2 {
					return nil, fmt.Errorf("invalid param: %s", v)
				}
				switch value := param[1]; param[0] {
				case "ssl-ca":
					ds.sslCA = value
				case "ssl-cert":
					ds.sslCert = value
				case "ssl-key":
					ds.sslKey = value
				default:
					ds.params[param[0]] = value
				}
			}
		}
	}

	return ds, nil
}
