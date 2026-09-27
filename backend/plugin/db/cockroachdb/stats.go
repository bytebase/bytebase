package cockroachdb

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// CountAffectedRows returns the optimizer's estimate of the rows the statement modifies. A search
// path that base.WithSearchPath put before the statement is set for the EXPLAIN.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	setup, statement := base.SplitSearchPath(statement)
	var settings []string
	if setup != "" {
		settings = append(settings, setup)
	}
	plan, err := d.explain(ctx, statement, settings...)
	if err != nil {
		return 0, err
	}
	rows, err := getAffectedRowsFromPlan(plan)
	if err == nil || !hasDeleteRangeNode(plan) {
		return rows, err
	}
	// A delete range node has no row estimate, while the plan of the same DELETE without the fast
	// path has. Versions without the setting keep the original error.
	fallbackPlan, fallbackErr := d.explain(ctx, statement, append(settings, "SET LOCAL optimizer_use_delete_range_fast_path = off")...)
	if fallbackErr != nil {
		return 0, err
	}
	return getAffectedRowsFromPlan(fallbackPlan)
}

// explain returns the EXPLAIN output of the statement after the SET LOCAL settings, which run in a
// transaction that is rolled back so the pooled connection keeps its session.
func (d *Driver) explain(ctx context.Context, statement string, settings ...string) ([]string, error) {
	var plan []string
	err := crdb.Execute(func() error {
		if len(settings) == 0 {
			var err error
			plan, err = queryPlan(ctx, d.db.QueryContext, statement)
			return err
		}
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, setting := range settings {
			if _, err := tx.ExecContext(ctx, setting); err != nil {
				return err
			}
		}
		plan, err = queryPlan(ctx, tx.QueryContext, statement)
		return err
	})
	return plan, err
}

func queryPlan(ctx context.Context, query func(context.Context, string, ...any) (*sql.Rows, error), statement string) ([]string, error) {
	rows, err := query(ctx, fmt.Sprintf("EXPLAIN %s", statement))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plan []string
	for rows.Next() {
		var line sql.NullString
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		plan = append(plan, line.String)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return plan, nil
}

type planNode struct {
	name string
	// column is the rune offset of the node's bullet, which grows with the node's depth.
	column     int
	attributes []string
}

// parsePlanNodes returns the nodes of EXPLAIN text output in pre-order. A node line has a bullet
// after the tree drawing, and the lines before the next node line are the node's attributes.
func parsePlanNodes(plan []string) []planNode {
	var nodes []planNode
	for _, line := range plan {
		line = strings.TrimRight(line, " \t\r")
		text := strings.TrimLeft(line, " │├└─")
		if name, ok := strings.CutPrefix(text, "• "); ok {
			nodes = append(nodes, planNode{
				name:   name,
				column: utf8.RuneCountInString(line[:len(line)-len(text)]),
			})
			continue
		}
		if len(nodes) > 0 && text != "" {
			nodes[len(nodes)-1].attributes = append(nodes[len(nodes)-1].attributes, text)
		}
	}
	return nodes
}

func hasDeleteRangeNode(plan []string) bool {
	return slices.ContainsFunc(parsePlanNodes(plan), func(node planNode) bool {
		return node.name == "delete range"
	})
}

var mutationNodeNames = map[string]bool{
	"insert":           true,
	"insert fast path": true,
	"upsert":           true,
	"update":           true,
	"delete":           true,
	"delete range":     true,
}

// getAffectedRowsFromPlan sums the rows written by the mutation nodes in EXPLAIN text output,
// including the mutations of CTEs and statement sources. Foreign key cascades are left out, as
// CockroachDB leaves them out of the rows a statement affects. It fails when the plan has no
// mutation node or a mutation node has no row count.
func getAffectedRowsFromPlan(plan []string) (int64, error) {
	nodes := parsePlanNodes(plan)
	var total float64
	mutationCount := 0
	cascadeColumn := -1
	for i, node := range nodes {
		if cascadeColumn >= 0 && node.column > cascadeColumn {
			continue
		}
		cascadeColumn = -1
		if node.name == "fk-cascade" {
			cascadeColumn = node.column
			continue
		}
		if !mutationNodeNames[node.name] {
			continue
		}
		rows, ok := getMutationRows(nodes, i)
		if !ok {
			return 0, errors.Errorf("the %s node in the plan has no row count estimate", node.name)
		}
		total += rows
		mutationCount++
	}
	if mutationCount == 0 {
		return 0, errors.New("the plan has no mutation node")
	}
	return common.RoundRows(total), nil
}

// getMutationRows returns the rows written by the mutation node nodes[i]. A mutation that returns
// rows has its own estimate. Otherwise the first row count in its subtree, in pre-order, estimates
// its input, capped by the counts of the limit nodes above it, which have no estimate of their own.
func getMutationRows(nodes []planNode, i int) (float64, bool) {
	limit := math.Inf(1)
	for j := i; j < len(nodes); j++ {
		node := nodes[j]
		if j > i && node.column <= nodes[i].column {
			break
		}
		for _, attribute := range node.attributes {
			key, value, _ := strings.Cut(attribute, ": ")
			switch {
			case key == "estimated row count":
				rows, ok := parseEstimatedRowCount(value)
				return math.Min(rows, limit), ok
			case key == "size" && (node.name == "values" || node.name == "insert fast path"):
				rows, ok := parseValuesSize(value)
				return math.Min(rows, limit), ok
			case key == "count" && node.name == "limit":
				if count, ok := parseCount(value); ok {
					limit = math.Min(limit, count)
				}
			default:
			}
		}
	}
	return 0, false
}

// parseEstimatedRowCount parses "1,000 (10% of the table; stats collected 1 minute ago)". A node
// under a soft limit shows a "min - max" range, whose maximum leaves the limit to the nodes above.
func parseEstimatedRowCount(value string) (float64, bool) {
	value, _, _ = strings.Cut(value, " (")
	if _, maximum, ok := strings.Cut(value, " - "); ok {
		value = maximum
	}
	return parseCount(value)
}

var valuesSizeRegexp = regexp.MustCompile(`^[\d,]+ columns?, ([\d,]+) rows?$`)

// parseValuesSize parses the rows of "5 columns, 2 rows".
func parseValuesSize(value string) (float64, bool) {
	match := valuesSizeRegexp.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}
	return parseCount(match[1])
}

func parseCount(value string) (float64, bool) {
	count, err := strconv.ParseUint(strings.ReplaceAll(value, ",", ""), 10, 64)
	if err != nil {
		return 0, false
	}
	return float64(count), true
}
