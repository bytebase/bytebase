// backend/component/export/download_goldens_test.go
package export

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

var updateGoldens = flag.Bool("update", false, "regenerate goldens under frontend/src/utils/sql-download/__tests__/goldens")

// goldensDir returns the absolute on-disk path of the frontend goldens
// directory. Test must be run from the repo root or any descendant.
func goldensDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "frontend", "src", "utils", "sql-download", "__tests__", "goldens")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found the repo root; create the directory.
			if err := os.MkdirAll(candidate, 0o755); err != nil {
				t.Fatalf("mkdir goldens: %v", err)
			}
			return candidate
		}
	}
	t.Fatal("could not locate repo root from working directory")
	return ""
}

// sqlEngines used for SQL goldens. Mirrors the spec dialect map.
var sqlEngines = []storepb.Engine{
	storepb.Engine_MYSQL,
	storepb.Engine_POSTGRES,
	storepb.Engine_TIDB,
	storepb.Engine_CLICKHOUSE,
	storepb.Engine_MSSQL,
	storepb.Engine_ORACLE,
	storepb.Engine_SNOWFLAKE,
	storepb.Engine_SQLITE,
	storepb.Engine_REDSHIFT,
	storepb.Engine_MARIADB,
	storepb.Engine_OCEANBASE,
	storepb.Engine_SPANNER,
}

// TestDownloadGoldens regenerates (with -update) or verifies (without)
// the per-format goldens consumed by frontend/src/utils/sql-download tests.
// Also writes goldens/fixture_ids.txt — a sorted manifest the TS test asserts
// against to catch fixture-map drift between Go and TS.
func TestDownloadGoldens(t *testing.T) {
	t.Parallel()
	dir := goldensDir(t)
	fixtures := downloadFixtures()

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.id, func(t *testing.T) {
			t.Parallel()
			runGolden(t, filepath.Join(dir, "csv", fx.id+".csv"), func() ([]byte, error) {
				return CSV(fx.result)
			})
			// Skip JSON for fixtures whose id ends with "_skip_json" — backend
			// would error out on NaN/Inf via encoding/json.
			if !strings.HasSuffix(fx.id, "_skip_json") {
				runGolden(t, filepath.Join(dir, "json", fx.id+".json"), func() ([]byte, error) {
					return JSON(fx.result)
				})
			}
			runGolden(t, filepath.Join(dir, "xlsx", fx.id+".xlsx"), func() ([]byte, error) {
				return XLSX(fx.result)
			})
			for _, eng := range sqlEngines {
				eng := eng
				prefix, err := SQLStatementPrefix(eng, nil, fx.result.ColumnNames)
				if err != nil {
					// Engine-fixture combinations that backend rejects (zero columns
					// usually) are skipped — frontend test mirrors the same skip.
					t.Logf("skip engine=%v fixture=%s: %v", eng, fx.id, err)
					continue
				}
				out := filepath.Join(dir, "sql", fx.id+"."+engineKey(eng)+".sql")
				runGolden(t, out, func() ([]byte, error) {
					return SQL(eng, prefix, fx.result)
				})
			}
		})
	}

	// Manifest of fixture ids. With -update, write; without, verify.
	ids := make([]string, 0, len(fixtures))
	for _, fx := range fixtures {
		ids = append(ids, fx.id)
	}
	slices.Sort(ids)
	manifest := strings.Join(ids, "\n") + "\n"
	manifestPath := filepath.Join(dir, "fixture_ids.txt")
	if *updateGoldens {
		if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	} else {
		got, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("read manifest %s (regenerate via -update): %v", manifestPath, err)
		}
		if string(got) != manifest {
			t.Fatalf(
				"fixture id manifest is stale: backend fixtures changed without regen. "+
					"Run: go test ./backend/component/export -run TestDownloadGoldens -update\n"+
					"want:\n%s\ngot:\n%s",
				manifest, string(got),
			)
		}
	}
}

func engineKey(e storepb.Engine) string {
	switch e {
	case storepb.Engine_MYSQL:
		return "mysql"
	case storepb.Engine_POSTGRES:
		return "postgres"
	case storepb.Engine_TIDB:
		return "tidb"
	case storepb.Engine_CLICKHOUSE:
		return "clickhouse"
	case storepb.Engine_MSSQL:
		return "mssql"
	case storepb.Engine_ORACLE:
		return "oracle"
	case storepb.Engine_SNOWFLAKE:
		return "snowflake"
	case storepb.Engine_SQLITE:
		return "sqlite"
	case storepb.Engine_REDSHIFT:
		return "redshift"
	case storepb.Engine_MARIADB:
		return "mariadb"
	case storepb.Engine_OCEANBASE:
		return "oceanbase"
	case storepb.Engine_SPANNER:
		return "spanner"
	}
	return "unknown"
}

func runGolden(t *testing.T, path string, gen func() ([]byte, error)) {
	t.Helper()
	got, err := gen()
	if err != nil {
		t.Fatalf("generate %s: %v", path, err)
	}
	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (regenerate with: go test ./backend/component/export -run TestDownloadGoldens -update): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s drift — backend output diverged from committed golden. Regenerate with: go test ./backend/component/export -run TestDownloadGoldens -update", path)
	}
}
