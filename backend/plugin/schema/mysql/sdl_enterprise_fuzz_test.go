package mysql

// A4 of the ENTERPRISE smoke axes (see sdl_enterprise_test.go): a SEEDED, deterministic
// random-migration fuzzer over a 30-table Zabbix slice. Each seed derives K ∈ [3,8]
// independent mutations from a menu (add/drop/modify column, add/drop index, add/drop FK
// with valid targets only, add/drop/modify view, add/drop trigger, add/drop routine,
// partition/departition), applies them TEXTUALLY to the canonical dump to build the
// target SDL, and runs the oracle protocol (diff → apply → converge → idempotence).
//
// Determinism: the mutation stream is a pure function of the seed (math/rand with a
// fixed source; the schema model is iterated in sorted order), so a failing seed replays
// exactly. Every failure prints the seed and a ddmin-minimized mutation list, each
// minimization trial re-running the full oracle on a fresh scratch database.
//
// ≥20 seeds run on 8.0 and ≥10 on 5.7 (the menu is 5.7-legal, so both use the same
// generator). Scratch databases are entsdl_-prefixed and dropped eagerly per trial.

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// entFuzzSliceTables is the ~30-table fuzz base: the core slice plus the httptest family
// (6 more triggered tables, FK-chained through httptest/httpstep into hosts and items)
// and token (a 2-FK fan-in on users).
var entFuzzSliceTables = append(append([]string{}, entSliceCoreTables...),
	"httptest", "httpstep", "httptestitem", "httpstepitem", "httptest_field", "httpstep_field",
	"token",
)

// entFuzzAux seeds the droppable/modifiable object pool the menu needs: two views, one
// function, one procedure, and a partitioned table (the departition target).
const entFuzzAux = `
CREATE TABLE ent_fz_part (
	id bigint unsigned NOT NULL,
	bucket integer NOT NULL,
	note varchar(40) DEFAULT '' NOT NULL,
	PRIMARY KEY (id,bucket)
) ENGINE=InnoDB
PARTITION BY HASH (bucket) PARTITIONS 4;
CREATE VIEW ent_fz_v1 AS SELECT userid, username FROM users;
CREATE VIEW ent_fz_v2 AS SELECT hostid, host FROM hosts;
CREATE FUNCTION ent_fz_fn1() RETURNS INT DETERMINISTIC RETURN 1;
CREATE PROCEDURE ent_fz_pr1() BEGIN SELECT 1; END;
`

// entFzProtectedTables are never touched by column-level mutations: the changelog
// trigger bodies name changelog's columns outright (not via NEW/OLD), and ent_fz_part
// belongs to the partition mutations.
var entFzProtectedTables = map[string]bool{"changelog": true, "ent_fz_part": true}

// entFzProtectedColumns are referenced by the seeded views (or their modify-view
// extension columns) and must survive column drops.
var entFzProtectedColumns = map[string]bool{
	"users.username": true, "users.name": true,
	"hosts.host": true, "hosts.status": true,
}

// ----------------------------------------------------------------------------
// Canonical-dump schema model (guides valid mutation choices; the mutations themselves
// are textual).
// ----------------------------------------------------------------------------

type entFzIndex struct {
	name   string
	cols   []string
	unique bool
}

type entFzFK struct {
	name string
	cols []string
}

type entFzTable struct {
	name        string
	cols        []string          // ordered
	colLine     map[string]string // name -> trimmed definition line
	pk          map[string]bool
	indexes     []entFzIndex
	fks         []entFzFK
	partitioned bool
}

type entFzModel struct {
	tables      []*entFzTable // sorted by name
	byName      map[string]*entFzTable
	referenced  map[string]bool            // tables referenced by any FK
	triggerCols map[string]map[string]bool // table -> NEW./OLD.-referenced columns
	triggers    []string                   // sorted trigger names
	views       []string                   // seeded, droppable views
	functions   []string                   // seeded, droppable functions
	procedures  []string                   // seeded, droppable procedures
}

var (
	entFzReColLine = regexp.MustCompile("^`(\\w+)` (.+?),?$")
	entFzRePK      = regexp.MustCompile(`^PRIMARY KEY \((.+)\),?$`)
	entFzReIndex   = regexp.MustCompile("^(UNIQUE )?KEY `(\\w+)` \\((.+)\\),?$")
	entFzReFK      = regexp.MustCompile("^CONSTRAINT `(\\w+)` FOREIGN KEY \\((.+?)\\) REFERENCES `(\\w+)`")
	entFzReTrigger = regexp.MustCompile("(?i)^CREATE TRIGGER `(\\w+)`.* ON `(\\w+)`")
	entFzReNewOld  = regexp.MustCompile(`(?i)\b(?:new|old)\.(\w+)`)
	entFzReCol     = regexp.MustCompile("`(\\w+)`")
	entFzReVarLen  = regexp.MustCompile(`varchar\((\d+)\)`)
)

// entFuzzModel parses the canonical dump into the mutation-planning model.
func entFuzzModel(t *testing.T, source string) *entFzModel {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(source)
	require.NoError(t, err, "split canonical dump for fuzz model")

	m := &entFzModel{
		byName:      map[string]*entFzTable{},
		referenced:  map[string]bool{},
		triggerCols: map[string]map[string]bool{},
	}
	for _, s := range stmts {
		text := strings.TrimSpace(s.Text)
		upper := strings.ToUpper(text)
		switch {
		case strings.HasPrefix(upper, "CREATE TABLE `"):
			tbl := entFzParseTable(text)
			if tbl != nil {
				m.tables = append(m.tables, tbl)
				m.byName[tbl.name] = tbl
			}
		case strings.HasPrefix(upper, "CREATE TRIGGER `") || (strings.HasPrefix(upper, "CREATE ") && strings.Contains(upper, " TRIGGER `")):
			if mm := entFzReTrigger.FindStringSubmatch(text); mm != nil {
				m.triggers = append(m.triggers, mm[1])
				onTable := mm[2]
				if m.triggerCols[onTable] == nil {
					m.triggerCols[onTable] = map[string]bool{}
				}
				for _, ref := range entFzReNewOld.FindAllStringSubmatch(text, -1) {
					m.triggerCols[onTable][ref[1]] = true
				}
			}
		default:
		}
	}
	// FK-referenced tables (for partition candidacy).
	for _, s := range stmts {
		for _, mm := range entFzReFKRefs.FindAllStringSubmatch(s.Text, -1) {
			m.referenced[mm[1]] = true
		}
	}
	// Seeded droppable objects (fixed names — the fuzz base owns them).
	m.views = []string{"ent_fz_v1", "ent_fz_v2"}
	m.functions = []string{"ent_fz_fn1"}
	m.procedures = []string{"ent_fz_pr1"}
	return m
}

var entFzReFKRefs = regexp.MustCompile("FOREIGN KEY \\(.+?\\) REFERENCES `(\\w+)`")

// entFzParseTable parses one canonical CREATE TABLE statement.
func entFzParseTable(stmt string) *entFzTable {
	header := regexp.MustCompile("^CREATE TABLE `(\\w+)`").FindStringSubmatch(stmt)
	if header == nil {
		return nil
	}
	tbl := &entFzTable{
		name:        header[1],
		colLine:     map[string]string{},
		pk:          map[string]bool{},
		partitioned: strings.Contains(strings.ToUpper(stmt), "PARTITION BY"),
	}
	for _, raw := range strings.Split(stmt, "\n")[1:] {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ")") {
			break
		}
		if mm := entFzRePK.FindStringSubmatch(line); mm != nil {
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[1], -1) {
				tbl.pk[c[1]] = true
			}
			continue
		}
		if mm := entFzReIndex.FindStringSubmatch(line); mm != nil {
			idx := entFzIndex{name: mm[2], unique: mm[1] != ""}
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[3], -1) {
				idx.cols = append(idx.cols, c[1])
			}
			tbl.indexes = append(tbl.indexes, idx)
			continue
		}
		if mm := entFzReFK.FindStringSubmatch(line); mm != nil {
			fk := entFzFK{name: mm[1]}
			for _, c := range entFzReCol.FindAllStringSubmatch(mm[2], -1) {
				fk.cols = append(fk.cols, c[1])
			}
			tbl.fks = append(tbl.fks, fk)
			continue
		}
		if strings.HasPrefix(line, "CONSTRAINT `") || strings.HasPrefix(line, "FULLTEXT KEY") {
			continue // CHECK constraints / fulltext — not mutation targets
		}
		if mm := entFzReColLine.FindStringSubmatch(line); mm != nil {
			tbl.cols = append(tbl.cols, mm[1])
			tbl.colLine[mm[1]] = strings.TrimSuffix(line, ",")
		}
	}
	return tbl
}

// entFzIndexedCols returns every column participating in any index or FK of tbl.
func entFzIndexedCols(tbl *entFzTable) map[string]bool {
	used := map[string]bool{}
	for c := range tbl.pk {
		used[c] = true
	}
	for _, idx := range tbl.indexes {
		for _, c := range idx.cols {
			used[c] = true
		}
	}
	for _, fk := range tbl.fks {
		for _, c := range fk.cols {
			used[c] = true
		}
	}
	return used
}

// ----------------------------------------------------------------------------
// Mutation generation.
// ----------------------------------------------------------------------------

// entFzMutation is one textual schema mutation with a replayable description.
type entFzMutation struct {
	desc  string
	apply func(t *testing.T, source string) string
}

// entFzState tracks per-round conflicts so the K mutations stay independent.
type entFzState struct {
	usedCols     map[string]bool // "table.col" touched by drop/modify
	usedIndexes  map[string]bool // "table.index" dropped
	usedFKs      map[string]bool
	usedViews    map[string]bool
	usedTriggers map[string]bool
	usedRoutines map[string]bool
	lockedTables map[string]bool // partition/departition/fk-structure locks
}

func entFzNewState() *entFzState {
	return &entFzState{
		usedCols:     map[string]bool{},
		usedIndexes:  map[string]bool{},
		usedFKs:      map[string]bool{},
		usedViews:    map[string]bool{},
		usedTriggers: map[string]bool{},
		usedRoutines: map[string]bool{},
		lockedTables: map[string]bool{},
	}
}

var entFzMenu = []string{
	"add_column", "drop_column", "modify_column",
	"add_index", "drop_index",
	"add_fk", "drop_fk",
	"add_view", "drop_view", "modify_view",
	"add_trigger", "drop_trigger",
	"add_routine", "drop_routine",
	"partition", "departition",
}

// entFuzzMutations derives k independent mutations from the model, deterministically in
// (model order, rng stream).
func entFuzzMutations(t *testing.T, model *entFzModel, rng *rand.Rand, k int) []entFzMutation {
	t.Helper()
	state := entFzNewState()
	var muts []entFzMutation
	for attempt := 0; attempt < 400 && len(muts) < k; attempt++ {
		kind := entFzMenu[rng.Intn(len(entFzMenu))]
		seq := len(muts)
		if m, ok := entFzGenerate(kind, model, rng, state, seq); ok {
			muts = append(muts, m)
		}
	}
	require.Len(t, muts, k, "fuzz generator starved (only %d of %d mutations)", len(muts), k)
	return muts
}

// entFzPickTable picks a random mutable (non-protected, non-locked) table.
func entFzPickTable(model *entFzModel, rng *rand.Rand, state *entFzState) *entFzTable {
	var candidates []*entFzTable
	for _, tbl := range model.tables {
		if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
			continue
		}
		candidates = append(candidates, tbl)
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rng.Intn(len(candidates))]
}

// entFzGenerate builds one mutation of the given kind, or reports it infeasible.
func entFzGenerate(kind string, model *entFzModel, rng *rand.Rand, state *entFzState, seq int) (entFzMutation, bool) {
	switch kind {
	case "add_column":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil {
			return entFzMutation{}, false
		}
		defs := []string{"varchar(40) DEFAULT NULL", "int NOT NULL DEFAULT '0'", "decimal(8,2) DEFAULT NULL", "bigint unsigned DEFAULT NULL"}
		def := defs[rng.Intn(len(defs))]
		col := fmt.Sprintf("ent_fz_c%d", seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_column %s.%s (%s)", table, col, def),
			apply: func(t *testing.T, source string) string {
				return addColumnToTable(t, source, table, "`"+col+"` "+def)
			},
		}, true

	case "drop_column":
		type cand struct{ table, col string }
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			indexed := entFzIndexedCols(tbl)
			trigRef := model.triggerCols[tbl.name]
			for _, c := range tbl.cols {
				if indexed[c] || trigRef[c] || entFzProtectedColumns[tbl.name+"."+c] {
					continue
				}
				if state.usedCols[tbl.name+"."+c] {
					continue
				}
				if strings.Contains(tbl.colLine[c], "GENERATED") {
					continue
				}
				cands = append(cands, cand{tbl.name, c})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_column %s.%s", pick.table, pick.col),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "`"+pick.col+"`")
			},
		}, true

	case "modify_column":
		type cand struct {
			table, col string
			from, to   int
		}
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			for _, c := range tbl.cols {
				if state.usedCols[tbl.name+"."+c] {
					continue
				}
				mm := entFzReVarLen.FindStringSubmatch(tbl.colLine[c])
				if mm == nil {
					continue
				}
				n := entFzAtoi(mm[1])
				if n < 8 || n > 150 {
					continue
				}
				cands = append(cands, cand{tbl.name, c, n, n + 37})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedCols[pick.table+"."+pick.col] = true
		return entFzMutation{
			desc: fmt.Sprintf("modify_column widen %s.%s varchar(%d)->varchar(%d)", pick.table, pick.col, pick.from, pick.to),
			apply: func(t *testing.T, source string) string {
				return entReplaceInTable(t, source, pick.table,
					fmt.Sprintf("`%s` varchar(%d)", pick.col, pick.from),
					fmt.Sprintf("`%s` varchar(%d)", pick.col, pick.to))
			},
		}, true

	case "add_index":
		type cand struct{ table, col string }
		var cands []cand
		for _, tbl := range model.tables {
			if entFzProtectedTables[tbl.name] || state.lockedTables[tbl.name] {
				continue
			}
			for _, c := range tbl.cols {
				line := tbl.colLine[c]
				rest := strings.TrimPrefix(line, "`"+c+"` ")
				ok := strings.HasPrefix(rest, "bigint") || strings.HasPrefix(rest, "int") || strings.HasPrefix(rest, "integer")
				if mm := entFzReVarLen.FindStringSubmatch(rest); mm != nil && strings.HasPrefix(rest, "varchar") {
					ok = entFzAtoi(mm[1]) <= 150
				}
				if ok {
					cands = append(cands, cand{tbl.name, c})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		idx := fmt.Sprintf("ent_fz_i%d", seq)
		return entFzMutation{
			desc: fmt.Sprintf("add_index %s on %s(%s)", idx, pick.table, pick.col),
			apply: func(t *testing.T, source string) string {
				return addIndexToTable(t, source, pick.table, "KEY `"+idx+"` (`"+pick.col+"`)")
			},
		}, true

	case "drop_index":
		type cand struct{ table, index string }
		var cands []cand
		for _, tbl := range model.tables {
			if state.lockedTables[tbl.name] {
				continue
			}
			fkCols := map[string]bool{}
			for _, fk := range tbl.fks {
				for _, c := range fk.cols {
					fkCols[c] = true
				}
			}
			for _, idx := range tbl.indexes {
				if state.usedIndexes[tbl.name+"."+idx.name] {
					continue
				}
				overlap := false
				for _, c := range idx.cols {
					if fkCols[c] {
						overlap = true
						break
					}
				}
				if !overlap {
					cands = append(cands, cand{tbl.name, idx.name})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedIndexes[pick.table+"."+pick.index] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_index %s.%s", pick.table, pick.index),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "`"+pick.index+"`")
			},
		}, true

	case "add_fk":
		src := entFzPickTable(model, rng, state)
		if src == nil || src.partitioned {
			return entFzMutation{}, false
		}
		var refs []*entFzTable
		for _, tbl := range model.tables {
			if tbl.partitioned || state.lockedTables[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			pkCol := ""
			for c := range tbl.pk {
				pkCol = c
			}
			if strings.HasPrefix(strings.TrimPrefix(tbl.colLine[pkCol], "`"+pkCol+"` "), "bigint") &&
				strings.Contains(tbl.colLine[pkCol], "unsigned") {
				refs = append(refs, tbl)
			}
		}
		if len(refs) == 0 {
			return entFzMutation{}, false
		}
		ref := refs[rng.Intn(len(refs))]
		refPK := ""
		for c := range ref.pk {
			refPK = c
		}
		srcName, refName := src.name, ref.name
		state.lockedTables[srcName] = true
		state.lockedTables[refName] = true
		col := fmt.Sprintf("ent_fz_r%d", seq)
		fk := fmt.Sprintf("ent_fz_fk%d", seq)
		return entFzMutation{
			desc: fmt.Sprintf("add_fk %s.%s -> %s.%s (%s)", srcName, col, refName, refPK, fk),
			apply: func(t *testing.T, source string) string {
				s := addColumnToTable(t, source, srcName, "`"+col+"` bigint unsigned DEFAULT NULL")
				s = addIndexToTable(t, s, srcName, "KEY `"+fk+"` (`"+col+"`)")
				return addIndexToTable(t, s, srcName,
					"CONSTRAINT `"+fk+"` FOREIGN KEY (`"+col+"`) REFERENCES `"+refName+"` (`"+refPK+"`) ON DELETE SET NULL")
			},
		}, true

	case "drop_fk":
		type cand struct{ table, fk string }
		var cands []cand
		for _, tbl := range model.tables {
			if state.lockedTables[tbl.name] {
				continue
			}
			for _, fk := range tbl.fks {
				if !state.usedFKs[tbl.name+"."+fk.name] {
					cands = append(cands, cand{tbl.name, fk.name})
				}
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedFKs[pick.table+"."+pick.fk] = true
		return entFzMutation{
			desc: fmt.Sprintf("drop_fk %s.%s", pick.table, pick.fk),
			apply: func(t *testing.T, source string) string {
				return entDropLineInTable(t, source, pick.table, "CONSTRAINT `"+pick.fk+"`")
			},
		}, true

	case "add_view":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil || len(tbl.pk) == 0 {
			return entFzMutation{}, false
		}
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entFzMutation{}, false
		}
		name := fmt.Sprintf("ent_fz_v%d", 10+seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_view %s over %s(%s)", name, table, pkCol),
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE VIEW %s AS SELECT %s FROM %s;\n", name, pkCol, table)
			},
		}, true

	case "drop_view":
		var cands []string
		for _, v := range model.views {
			if !state.usedViews[v] {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		return entFzMutation{
			desc: "drop_view " + pick,
			apply: func(t *testing.T, source string) string {
				return dropObjectBlock(t, source, "VIEW", pick)
			},
		}, true

	case "modify_view":
		extension := map[string]string{
			"ent_fz_v1": "`users`.`name` AS `name`",
			"ent_fz_v2": "`hosts`.`status` AS `status`",
		}
		var cands []string
		for _, v := range model.views {
			if !state.usedViews[v] && extension[v] != "" {
				cands = append(cands, v)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedViews[pick] = true
		ext := extension[pick]
		return entFzMutation{
			desc: "modify_view " + pick,
			apply: func(t *testing.T, source string) string {
				return addColumnToView(t, source, pick, ext)
			},
		}, true

	case "add_trigger":
		tbl := entFzPickTable(model, rng, state)
		if tbl == nil || len(tbl.pk) == 0 {
			return entFzMutation{}, false
		}
		pkCol := ""
		for _, c := range tbl.cols {
			if tbl.pk[c] {
				pkCol = c
				break
			}
		}
		if pkCol == "" {
			return entFzMutation{}, false
		}
		name := fmt.Sprintf("ent_fz_t%d", seq)
		table := tbl.name
		return entFzMutation{
			desc: fmt.Sprintf("add_trigger %s on %s", name, table),
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE TRIGGER %s BEFORE UPDATE ON %s FOR EACH ROW SET NEW.%s = NEW.%s;\n",
					name, table, pkCol, pkCol)
			},
		}, true

	case "drop_trigger":
		var cands []string
		for _, trg := range model.triggers {
			if !state.usedTriggers[trg] {
				cands = append(cands, trg)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedTriggers[pick] = true
		return entFzMutation{
			desc: "drop_trigger " + pick,
			apply: func(t *testing.T, source string) string {
				return dropTrigger(t, source, pick)
			},
		}, true

	case "add_routine":
		if rng.Intn(2) == 0 {
			name := fmt.Sprintf("ent_fz_fn%d", 10+seq)
			ret := seq + 100
			return entFzMutation{
				desc: "add_routine function " + name,
				apply: func(_ *testing.T, source string) string {
					return source + fmt.Sprintf("\nCREATE FUNCTION %s() RETURNS INT DETERMINISTIC RETURN %d;\n", name, ret)
				},
			}, true
		}
		name := fmt.Sprintf("ent_fz_pr%d", 10+seq)
		sel := seq + 200
		return entFzMutation{
			desc: "add_routine procedure " + name,
			apply: func(_ *testing.T, source string) string {
				return source + fmt.Sprintf("\nCREATE PROCEDURE %s() BEGIN SELECT %d; END;\n", name, sel)
			},
		}, true

	case "drop_routine":
		type cand struct{ kind, name string }
		var cands []cand
		for _, fn := range model.functions {
			if !state.usedRoutines[fn] {
				cands = append(cands, cand{"FUNCTION", fn})
			}
		}
		for _, pr := range model.procedures {
			if !state.usedRoutines[pr] {
				cands = append(cands, cand{"PROCEDURE", pr})
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.usedRoutines[pick.name] = true
		return entFzMutation{
			desc: "drop_routine " + strings.ToLower(pick.kind) + " " + pick.name,
			apply: func(t *testing.T, source string) string {
				return dropObjectBlock(t, source, pick.kind, pick.name)
			},
		}, true

	case "partition":
		var cands []*entFzTable
		for _, tbl := range model.tables {
			// changelog is column-protected but IS the partition candidate (the only
			// FK-free, unreferenced, single-PK table in the slice); ent_fz_part is
			// already partitioned.
			if tbl.partitioned || state.lockedTables[tbl.name] {
				continue
			}
			if len(tbl.fks) > 0 || model.referenced[tbl.name] || len(tbl.pk) != 1 {
				continue
			}
			// The partition key must be part of every unique key: single-col-PK tables
			// with no other UNIQUE index qualify.
			hasUnique := false
			for _, idx := range tbl.indexes {
				if idx.unique {
					hasUnique = true
					break
				}
			}
			if hasUnique {
				continue
			}
			cands = append(cands, tbl)
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		pkCol := ""
		for c := range pick.pk {
			pkCol = c
		}
		state.lockedTables[pick.name] = true
		table := pick.name
		return entFzMutation{
			desc: fmt.Sprintf("partition %s by hash(%s)", table, pkCol),
			apply: func(t *testing.T, source string) string {
				return entAppendTableClause(t, source, table, "PARTITION BY HASH (`"+pkCol+"`)\nPARTITIONS 4")
			},
		}, true

	case "departition":
		var cands []*entFzTable
		for _, tbl := range model.tables {
			if tbl.partitioned && !state.lockedTables[tbl.name] {
				cands = append(cands, tbl)
			}
		}
		if len(cands) == 0 {
			return entFzMutation{}, false
		}
		pick := cands[rng.Intn(len(cands))]
		state.lockedTables[pick.name] = true
		table := pick.name
		return entFzMutation{
			desc: "departition " + table,
			apply: func(t *testing.T, source string) string {
				return entSetPartitionClause(t, source, table, "")
			},
		}, true

	default:
		return entFzMutation{}, false
	}
}

// entFzAtoi is a no-error atoi for regex-captured digits.
func entFzAtoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

// ----------------------------------------------------------------------------
// Trial runner + ddmin minimization.
// ----------------------------------------------------------------------------

// entFuzzScratchDB creates an eagerly-droppable scratch database (fuzz + minimization
// churn through many databases inside a single test, so cleanup cannot wait for test
// end).
func entFuzzScratchDB(ctx context.Context, t *testing.T, srv liveServer) (string, func()) {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, "entsdl_fz")
	drop := func() {
		c, err := createLiveMySQLDriver(ctx, srv, "")
		if err != nil {
			return
		}
		defer c.Close(ctx)
		_, _ = c.Execute(ctx, "DROP DATABASE IF EXISTS `"+dbName+"`", db.ExecuteOptions{})
	}
	return dbName, drop
}

// entFuzzTrial runs the whole oracle protocol for one mutation set on a fresh scratch
// database. Infrastructure failures (server down, base DDL broken, mutation anchors
// missing) fail the test hard; ORACLE violations (diff error, apply error, residual
// after apply, non-idempotent dump) are returned as errors so the caller can minimize.
func entFuzzTrial(ctx context.Context, t *testing.T, srv liveServer, baseDDL string, muts []entFzMutation) error {
	t.Helper()
	dbName, drop := entFuzzScratchDB(ctx, t, srv)
	defer drop()

	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(baseDDL), db.ExecuteOptions{})
	driver.Close(ctx)
	require.NoError(t, err, "[%s] fuzz base failed to load", srv.name)

	source := dumpSDL(ctx, t, srv, dbName)
	target := source
	for _, m := range muts {
		target = m.apply(t, target)
	}

	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	if err != nil {
		return errors.Wrap(err, "diff failed")
	}
	if strings.TrimSpace(plan) == "" {
		return errors.New("empty plan for a non-empty mutation set")
	}
	if err := applyDDL(ctx, t, srv, dbName, plan); err != nil {
		return errors.Wrapf(err, "plan failed to apply; plan was:\n%s", plan)
	}
	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, srv.version)
	if err != nil {
		return errors.Wrap(err, "converge diff failed")
	}
	if converge != "" {
		return errors.Errorf("did not converge; residual:\n%s\nplan was:\n%s", converge, plan)
	}
	self, err := mysqlDiffSDLMigration(after, after, srv.version)
	if err != nil {
		return errors.Wrap(err, "idempotence diff failed")
	}
	if self != "" {
		return errors.Errorf("post-apply dump not idempotent:\n%s", self)
	}
	return nil
}

// entFuzzMinimize greedily shrinks a failing mutation set (ddmin-lite: drop one mutation
// at a time, keep the removal whenever the rest still fails). Returns the minimized set
// and its error.
func entFuzzMinimize(ctx context.Context, t *testing.T, srv liveServer, baseDDL string, muts []entFzMutation, firstErr error) ([]entFzMutation, error) {
	t.Helper()
	minimized := muts
	lastErr := firstErr
	for i := 0; i < len(minimized) && len(minimized) > 1; {
		trial := make([]entFzMutation, 0, len(minimized)-1)
		trial = append(trial, minimized[:i]...)
		trial = append(trial, minimized[i+1:]...)
		if err := entFuzzTrial(ctx, t, srv, baseDDL, trial); err != nil {
			minimized = trial
			lastErr = err
			continue // same index now names the next mutation
		}
		i++
	}
	return minimized, lastErr
}

func entFzDescs(muts []entFzMutation) []string {
	out := make([]string, len(muts))
	for i, m := range muts {
		out[i] = m.desc
	}
	return out
}

// ----------------------------------------------------------------------------
// The fuzz test: ≥20 seeds on 8.0, ≥10 on 5.7 (deterministic; the seed is in the
// subtest name).
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseFuzz(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		seedCount := 20
		if srv.version == "5.7" {
			seedCount = 10
		}
		t.Run(srv.name, func(t *testing.T) {
			baseDDL := entZabbixSlice(t, entFuzzSliceTables...) + entFuzzAux

			// The model is built once per server from a reference load (the canonical dump
			// is deterministic, so it matches every trial's source).
			refDB, refDrop := entFuzzScratchDB(ctx, t, srv)
			driver, err := createLiveMySQLDriver(ctx, srv, refDB)
			require.NoError(t, err)
			_, err = driver.Execute(ctx, entNormalizeDelimiters(baseDDL), db.ExecuteOptions{})
			driver.Close(ctx)
			require.NoError(t, err, "[%s] fuzz base failed to load", srv.name)
			refSource := dumpSDL(ctx, t, srv, refDB)
			refDrop()
			model := entFuzzModel(t, refSource)
			require.NotEmpty(t, model.tables, "fuzz model parsed no tables")
			require.NotEmpty(t, model.triggers, "fuzz model parsed no triggers")

			for seed := int64(1); seed <= int64(seedCount); seed++ {
				seed := seed
				t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
					rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fuzz, not crypto
					k := 3 + rng.Intn(6)                  // K ∈ [3,8]
					muts := entFuzzMutations(t, model, rng, k)
					t.Logf("[FUZZ %s seed=%d] K=%d mutations:\n  %s", srv.name, seed, k, strings.Join(entFzDescs(muts), "\n  "))

					err := entFuzzTrial(ctx, t, srv, baseDDL, muts)
					if err == nil {
						return
					}
					minimized, minErr := entFuzzMinimize(ctx, t, srv, baseDDL, muts, err)
					t.Errorf("[FUZZ %s seed=%d] FAILED\nminimized mutations (%d of %d):\n  %s\nerror:\n%v",
						srv.name, seed, len(minimized), len(muts), strings.Join(entFzDescs(minimized), "\n  "), minErr)
				})
			}
		})
	}
}
