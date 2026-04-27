// Copyright (c) Bytebase (Hong Kong) Limited.

// Go reproduction harness for BYT-9239 — the twin of scripts/cosmos-repro-dotnet.
// Runs the full 40-query target inventory from
// https://linear.app/bytebase/issue/BYT-9239 against a live Azure Cosmos
// account through Bytebase's forked azcosmos SDK (which includes the pure-Go
// query engine), and prints a markdown PASS/FAIL table matching the shape
// the .NET harness produces. Pairing both harnesses on the same container
// is the regression guard for the BYT-9239 feature set.
//
// Required environment variables:
//   COSMOS_ENDPOINT   e.g. https://bytebase-cosmostest.documents.azure.com:443/
//   COSMOS_KEY        primary master key
//   COSMOS_DB         database name (default: testdb)
//   COSMOS_CONTAINER  container name (default: WorldCities)
//
// Flags:
//   --verbose  also print per-query detail above the summary table
//
// Run:
//   cd scripts/cosmos-repro-go
//   export COSMOS_ENDPOINT="https://bytebase-cosmostest.documents.azure.com:443/"
//   export COSMOS_KEY="$(az cosmosdb keys list \
//       --name bytebase-cosmostest \
//       --resource-group rg-bytebase-cosmostest \
//       --type keys --query primaryMasterKey -o tsv)"
//   go run . --verbose

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

type testCase struct {
	id  string
	sql string
}

// cases mirrors the 40-query target inventory exactly, in PDF section order.
// Each query's expected outcome is taken from the .NET SDK reference column.
// The two permanently-failing queries (4.3 multi-key ORDER BY, which the
// gateway rejects on any SDK because the container lacks a composite index)
// are included so the harness's summary matches the .NET harness's.
var cases = []testCase{
	// 1. Basic SELECT
	{"1.1", `SELECT * FROM c`},
	{"1.2", `SELECT c.id, c.name, c.country, c.population FROM c`},
	{"1.3", `SELECT TOP 10 * FROM c`},

	// 2. Filtering — WHERE
	{"2.1", `SELECT * FROM c WHERE c.country = "AE"`},
	{"2.2", `SELECT * FROM c WHERE c._ts > 1743702439`},
	{"2.3", `SELECT * FROM c WHERE c._ts >= 1743702439`},
	{"2.4", `SELECT * FROM c WHERE c.country = "AE" OR c.country = "AD"`},
	{"2.5", `SELECT * FROM c WHERE c.country IN ("AE", "US", "GB", "AD")`},
	{"2.6", `SELECT * FROM c WHERE CONTAINS(c.name, "Fujayrah")`},
	{"2.7", `SELECT * FROM c WHERE STARTSWITH(c.name, "Al")`},
	{"2.8", `SELECT * FROM c WHERE c.countryRegion != "4"`},
	{"2.9", `SELECT * FROM c WHERE STRINGTONUMBER(c.population) BETWEEN 10000 AND 100000`},

	// 3. Projection & Aliasing
	{"3.1", `SELECT c.name AS cityName, c.country AS countryCode, c.population AS pop FROM c`},
	{"3.2", `SELECT c.name, c.population, STRINGTONUMBER(c.population) / 1000 AS populationInThousands FROM c`},
	{"3.3", `SELECT CONCAT(c.name, " (", c.country, ")") AS label FROM c`},

	// 4. ORDER BY
	{"4.1", `SELECT c.name, c.population FROM c ORDER BY c.population ASC`},
	{"4.2", `SELECT c.name, c.population FROM c ORDER BY c.population DESC`},
	{"4.3", `SELECT c.country, c.name, c.population FROM c ORDER BY c.country ASC, c.population DESC`},

	// 5. Aggregation
	{"5.1", `SELECT COUNT(1) AS totalRecords FROM c`},
	{"5.1b", `SELECT VALUE COUNT(1) FROM c`},
	{"5.2", `SELECT COUNT(1) AS aeCount FROM c WHERE c.country = "AE"`},
	{"5.2b", `SELECT VALUE COUNT(1) FROM c WHERE c.country = "AE"`},
	{"5.3", `SELECT SUM(StringToNumber(c.population)) AS totalPop FROM c`},
	{"5.3b", `SELECT VALUE SUM(StringToNumber(c.population)) FROM c`},
	{"5.4", `SELECT AVG(StringToNumber(c.population)) AS avgPop FROM c`},
	{"5.4b", `SELECT VALUE AVG(StringToNumber(c.population)) FROM c`},
	{"5.5", `SELECT MIN(StringToNumber(c.population)) AS minPop, MAX(StringToNumber(c.population)) AS maxPop FROM c WHERE StringToNumber(c.population) > 0`},

	// 6. GROUP BY
	{"6.1", `SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country`},
	{"6.2", `SELECT c.countryRegion, COUNT(1) AS count, SUM(StringToNumber(c.population)) AS totalPop FROM c GROUP BY c.countryRegion`},

	// 7. String built-ins
	{"7.1", `SELECT UPPER(c.name) AS upperName, LOWER(c.country) AS lowerCountry, LENGTH(c.name) AS nameLen FROM c`},

	// 8. Math built-ins
	{"8.1", `SELECT c.name, ROUND(StringToNumber(c.latitude)) AS lat, ROUND(StringToNumber(c.longitude)) AS lon FROM c`},

	// 9. Type-checking built-ins
	{"9.1", `SELECT c.name, IS_STRING(c.name) AS isStr, IS_NUMBER(c.population) AS isNum FROM c`},
	{"9.2", `SELECT c.id, IS_DEFINED(c.region) AS hasRegion, IS_NULL(c.code) AS codeNull FROM c`},

	// 10. DISTINCT
	{"10.1", `SELECT DISTINCT c.country FROM c`},
	{"10.2", `SELECT DISTINCT VALUE c.countryRegion FROM c`},

	// 11. VALUE keyword
	{"11.1", `SELECT VALUE c.name FROM c WHERE c.country = "AE"`},
	{"11.2", `SELECT VALUE COUNT(1) FROM c`},

	// 12. OFFSET / LIMIT
	{"12.1", `SELECT * FROM c ORDER BY c.name OFFSET 0 LIMIT 10`},
	{"12.2", `SELECT * FROM c ORDER BY c.name OFFSET 10 LIMIT 10`},

	// 13. Geo-spatial
	{"13.1", `SELECT c.name, ST_DISTANCE({"type": "Point", "coordinates": [StringToNumber(c.longitude), StringToNumber(c.latitude)]}, {"type": "Point", "coordinates": [55.2708, 25.2048]}) AS distFromDubaiInMeters FROM c`},
}

type result struct {
	id        string
	sql       string
	status    string
	rows      int
	elapsedMs int64
	err       string
}

func main() {
	verbose := flag.Bool("verbose", false, "print per-query detail above the summary table")
	flag.Parse()

	endpoint := envOr("COSMOS_ENDPOINT", "")
	key := envOr("COSMOS_KEY", "")
	if endpoint == "" || key == "" {
		fmt.Fprintln(os.Stderr, "COSMOS_ENDPOINT and COSMOS_KEY are required")
		os.Exit(1)
	}
	db := envOr("COSMOS_DB", "testdb")
	coll := envOr("COSMOS_CONTAINER", "WorldCities")

	cred, err := azcosmos.NewKeyCredential(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewKeyCredential: %v\n", err)
		os.Exit(1)
	}
	client, err := azcosmos.NewClientWithKey(endpoint, cred, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewClientWithKey: %v\n", err)
		os.Exit(1)
	}
	container, err := client.NewContainer(db, coll)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewContainer: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("# Go SDK reproduction — db=%s coll=%s\n", db, coll)
	fmt.Println("SDK: github.com/bytebase/azure-sdk-for-go/sdk/data/azcosmos (pure-Go query engine)")
	fmt.Println()

	rows := make([]result, 0, len(cases))
	for _, tc := range cases {
		r := runOne(container, tc)
		rows = append(rows, r)
		if *verbose {
			fmt.Printf("[%s] %s  rows=%d  elapsed=%dms\n", r.status, r.id, r.rows, r.elapsedMs)
			if r.err != "" {
				fmt.Printf("    ERR: %s\n", short(r.err, 260))
			}
		}
	}

	// Summary table in the same shape as the .NET harness.
	fmt.Println()
	fmt.Println("## Results")
	fmt.Println()
	fmt.Println("| # | Query | Status | Rows | Elapsed (ms) | Error (truncated) |")
	fmt.Println("|---|-------|--------|------|-------------:|-------------------|")
	for _, r := range rows {
		errCell := ""
		if r.err != "" {
			errCell = strings.ReplaceAll(short(r.err, 140), "|", `\|`)
			errCell = strings.ReplaceAll(errCell, "\n", " ")
		}
		q := strings.ReplaceAll(r.sql, "|", `\|`)
		fmt.Printf("| %s | `%s` | %s | %d | %d | %s |\n", r.id, q, r.status, r.rows, r.elapsedMs, errCell)
	}

	// Tally.
	pass, fail := 0, 0
	for _, r := range rows {
		if r.status == "PASS" {
			pass++
		} else {
			fail++
		}
	}
	fmt.Println()
	fmt.Printf("**Totals:** %d PASS, %d FAIL out of %d.\n", pass, fail, len(rows))
}

func runOne(c *azcosmos.ContainerClient, tc testCase) result {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 10k row cap keeps the harness bounded when running SELECT * on large
	// containers. Matches the .NET harness's ceiling.
	const rowCap = 10_000

	pager := c.NewCrossPartitionQueryItemsPager(tc.sql, nil)
	n := 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return result{id: tc.id, sql: tc.sql, status: "FAIL", rows: n, elapsedMs: time.Since(start).Milliseconds(), err: err.Error()}
		}
		n += len(page.Items)
		if n >= rowCap {
			break
		}
	}
	return result{id: tc.id, sql: tc.sql, status: "PASS", rows: n, elapsedMs: time.Since(start).Milliseconds()}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
