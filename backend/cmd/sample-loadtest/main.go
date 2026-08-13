// Command sample-loadtest runs the sample-instance load test against an
// arbitrary PostgreSQL instance (local or GCP Cloud SQL) via an admin DSN.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/loadtest"
	"github.com/bytebase/bytebase/backend/resources/postgres"
)

func main() {
	host := flag.String("host", "127.0.0.1", "PostgreSQL host")
	port := flag.Int("port", 5432, "PostgreSQL port")
	user := flag.String("user", "postgres", "admin user with CREATEDB and CREATEROLE")
	password := flag.String("password", "", "admin password")
	sslmode := flag.String("sslmode", "disable", "pgx sslmode (disable|require|verify-full)")
	counts := flag.String("counts", "70,500,1000", "comma-separated database counts")
	concurrencies := flag.String("concurrencies", "10,50", "comma-separated interactive concurrency levels")
	report := flag.String("report", "", "output path for the JSON report (empty = no file)")
	flag.Parse()

	if *password == "" {
		log.Fatal("--password is required")
	}

	dbCounts, err := parseInts(*counts)
	if err != nil {
		log.Fatal(err)
	}
	concurrencyLevels, err := parseInts(*concurrencies)
	if err != nil {
		log.Fatal(err)
	}

	seed, err := postgres.LoadSampleData()
	if err != nil {
		log.Fatalf("load sample data: %v", err)
	}

	cfg := loadtest.Config{
		Host:                     *host,
		Port:                     *port,
		AdminUser:                *user,
		AdminPassword:            *password,
		SSLMode:                  *sslmode,
		DatabaseNamePrefix:       "lt_ws_",
		RoleNamePrefix:           "lt_role_",
		SeedSQL:                  seed,
		DatabaseCounts:           dbCounts,
		InteractiveConcurrencies: concurrencyLevels,
		SyncConcurrency:          100,
		InteractiveQueries:       loadtest.DefaultInteractiveQueries(),
		DDLStatements:            loadtest.DefaultDDLStatements(),
		ReportPath:               *report,
	}

	results, err := loadtest.Run(context.Background(), cfg)
	if err != nil {
		log.Fatalf("load test failed: %v", err)
	}
	fmt.Printf("completed %d database-count runs\n", len(results))
}

func parseInts(s string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(s, ",") {
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid integer %q", part)
		}
		out = append(out, n)
	}
	return out, nil
}
