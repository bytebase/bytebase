import { createContext, type ReactNode, useContext } from "react";
import type { PlanWarning } from "./plan-model";

const ENGLISH = {
  "copy.copied": "Copied",
  "copy.failed": "Copy failed",
  "copy.plan": "Copy plan",
  "details.select-node": "Select a node to see its details.",
  "diagram.actual-rows": "actual rows {{count}}",
  "diagram.collapse": "Collapse {{count}} nodes under {{node}}",
  "diagram.cost": "cost {{cost}}",
  "diagram.estimated-rows": "{{count}} estimated rows",
  "diagram.expand": "Expand {{count}} nodes under {{node}}",
  "diagram.fit": "Fit to view",
  "diagram.nodes-hidden": "{{count}} nodes hidden",
  "diagram.plan-cost-share": "{{share}} of plan cost",
  "diagram.plan-time-share": "{{share}} of plan time",
  "diagram.rows": "rows {{count}}",
  "diagram.time": "time {{time}} ms",
  "diagram.zoom-in": "Zoom in",
  "diagram.zoom-out": "Zoom out",
  "error.mssql-empty": "SQL Server returned no query plan for this statement.",
  "error.mssql-invalid":
    "The query plan is not valid XML. SHOWPLAN_XML output is expected.",
  "error.mssql-no-plan":
    "The query plan XML is not a SQL Server showplan with at least one statement.",
  "error.clickhouse-empty":
    "ClickHouse returned no query plan for this statement.",
  "error.clickhouse-invalid":
    "The query plan is not valid JSON. EXPLAIN json = 1 output is expected.",
  "error.clickhouse-no-plan":
    'The ClickHouse query plan JSON does not contain a "Plan" object.',
  "error.cockroachdb-empty":
    "CockroachDB returned no query plan for this statement.",
  "error.cockroachdb-invalid":
    "The query plan is not valid JSON. The diagram EXPLAIN (DISTSQL) encodes is expected.",
  "error.cockroachdb-no-plan":
    'The query plan JSON contains no "processors" to draw.',
  "error.mongodb-empty": "MongoDB returned no query plan for this statement.",
  "error.mongodb-invalid":
    "The query plan is not valid JSON. MongoDB's explain() output is expected.",
  "error.mongodb-no-plan":
    'The query plan JSON has neither a "queryPlanner" nor a "stages" pipeline to draw.',
  "error.mysql-empty": "No query plan came back for this statement.",
  "error.mysql-invalid":
    "The query plan is not valid JSON. EXPLAIN FORMAT=JSON output is expected.",
  "error.mysql-no-plan":
    'The query plan JSON does not contain a "query_block" object.',
  "error.oceanbase-empty":
    "OceanBase returned no query plan for this statement.",
  "error.oceanbase-invalid":
    "The query plan is not valid JSON. OceanBase's EXPLAIN FORMAT=JSON output is expected.",
  "error.oceanbase-no-plan":
    "The query plan JSON names no operator, so there is nothing to draw.",
  "error.oracle-empty": "Oracle returned no query plan for this statement.",
  "error.oracle-invalid":
    "The query plan is not valid JSON. Oracle's plan table is expected, as the server renders it.",
  "error.oracle-no-plan":
    "The query plan has no step without a parent, so it has no operator to start from.",
  "error.postgres-empty":
    "PostgreSQL returned no query plan for this statement.",
  "error.postgres-invalid":
    "The query plan is not valid JSON. EXPLAIN (FORMAT JSON) output is expected.",
  "error.postgres-no-plan":
    'The query plan JSON does not contain a "Plan" object.',
  "error.redshift-empty": "Redshift returned no query plan for this statement.",
  "error.redshift-no-plan":
    "The query plan has no operator in it: a Redshift plan states a cost for every one.",
  "error.spanner-emulator":
    "Spanner returned no query plan for this statement. The Spanner emulator returns none for any statement, so explain it on a Spanner instance to see one.",
  "error.spanner-empty": "Spanner returned no query plan for this statement.",
  "error.spanner-invalid":
    "The query plan is not valid JSON. A Spanner query plan is expected.",
  "error.spanner-no-plan":
    'The query plan JSON does not contain a "planNodes" list with an operator at its root.',
  "error.starrocks-empty": "The statement produced no query plan to draw.",
  "error.starrocks-no-plan":
    "The query plan has no numbered operator in any of its fragments.",
  "error.tidb-empty": "TiDB returned no query plan for this statement.",
  "error.tidb-invalid":
    "The query plan is not valid JSON. EXPLAIN FORMAT=tidb_json output is expected.",
  "error.tidb-no-plan":
    "The query plan JSON does not contain an operator at its root.",
  "error.trino-empty": "Trino returned no query plan for this statement.",
  "error.trino-invalid":
    "The query plan is not valid JSON. Trino's EXPLAIN (FORMAT JSON) output is expected.",
  "error.trino-no-plan":
    "The query plan JSON contains no plan fragment to start from.",
  "error.too-deep":
    "The query plan is nested too deeply for this visualizer to draw.",
  "error.too-many-nodes":
    "The query plan has more operators than this visualizer draws, so it is shown as text.",
  "error.unreadable": "This query plan could not be read",
  "grid.actual-rows": "Actual rows",
  "grid.actual-rows-hint":
    "Rows the operator really returned, over all its runs together",
  "grid.added-cost": "Added cost",
  "grid.added-cost-hint":
    "Cost this node adds on top of its children; some engines also count an input's repeats here, as under a nested loop. The bar is that cost as a share of the plan.",
  "grid.caption":
    "Plan nodes in the order the planner nests them, parents before children.",
  "grid.estimated-rows": "Estimated rows",
  "grid.node": "Node",
  "grid.rows": "Rows",
  "grid.self-time": "Self time",
  "grid.self-time-hint":
    "Milliseconds this node took on top of its children, over all its runs. The bar is that time as a share of the plan.",
  "grid.total-cost": "Total cost",
  "grid.width": "Width",
  "grid.width-hint": "Estimated row width in bytes",
  "highlight.cost": "Cost",
  "highlight.cost-hint": "Shade by the cost a node adds on top of its children",
  "highlight.label": "Highlight",
  "highlight.off": "Off",
  "highlight.off-hint": "Leave every node card unshaded",
  "highlight.rows": "Rows",
  "highlight.rows-hint": "Shade by the estimated row count",
  "highlight.time": "Time",
  "highlight.time-hint": "Shade by the time a node took on top of its children",
  "legend.cost":
    "each card's bar is that node's share of the plan's estimated cost",
  "legend.rows":
    "an edge thickens with the rows its child is estimated to return",
  "legend.time":
    "each card's bar is that node's share of the plan's measured time",
  "metric.actual-rows": "Actual rows",
  "metric.actual-time": "Actual time",
  "metric.added-cost": "Added cost",
  "metric.bytes": "{{count}} bytes",
  "metric.estimated-rows": "Estimated rows",
  "metric.loops": "Loops",
  "metric.milliseconds": "{{count}} ms",
  "metric.row-width": "Row width",
  "metric.rows-none": "Returned no rows, where the planner expected some.",
  "metric.rows-over":
    "Returned {{factor}}× more rows than the planner estimated.",
  "metric.rows-under":
    "Returned {{factor}}× fewer rows than the planner estimated.",
  "metric.startup-cost": "Startup cost",
  "metric.total-cost": "Total cost",
  "summary.costliest": "Costliest by estimated cost",
  "summary.no-added-cost":
    "No node in this plan adds an estimated cost of its own.",
  "summary.no-added-time":
    "No node in this plan took measurable time of its own.",
  "summary.no-ranking":
    "This plan reports no cost or time, so its operators cannot be ranked.",
  "summary.slowest-measured": "Slowest by measured time",
  "summary.warning-operators": "{{count}} operators",
  "summary.warnings": "Warnings",
  "tab.diagram": "Diagram",
  "tab.grid": "Table",
  "tab.text": "Text",
  "table.heading": "Table",
  "table.indexes": "Indexes",
  "table.no-indexes": "No indexes on this table.",
  "table.primary": "primary",
  "table.rows": "{{count}} rows at the last sync",
  "table.unique": "unique",
  "totals.estimated-cost": "estimated cost {{cost}}",
  "totals.measured-time": "measured time {{time}} ms",
  "totals.nodes": "{{count}} nodes",
  "warning.data-redistribution.detail":
    "This join moves rows between slices before it can match them. Distributing both tables on their join column, or making the smaller one ALL, lets the join run where the rows already are.",
  "warning.data-redistribution.title": "Data redistribution",
  "warning.full-index-scan.detail":
    "Reads every entry of this index. If the query needs only a few of them, a condition on the index's leading columns could avoid the full read.",
  "warning.full-index-scan.title": "Full index scan",
  "warning.full-table-scan.detail":
    "Reads every row of this table. If the query needs only a few of them, an index on the filtered columns could avoid the full read.",
  "warning.full-table-scan.title": "Full table scan",
} as const;

export type QueryPlanTranslationKey = keyof typeof ENGLISH;
export type QueryPlanTranslationValues = Record<string, string | number>;
export type QueryPlanTranslate = (
  key: QueryPlanTranslationKey,
  values?: QueryPlanTranslationValues
) => string;

const interpolate = (template: string, values?: QueryPlanTranslationValues) =>
  template.replace(/{{(\w+)}}/g, (_, key: string) =>
    String(values?.[key] ?? "")
  );

const fallbackTranslate: QueryPlanTranslate = (key, values) =>
  interpolate(ENGLISH[key], values);

const QueryPlanI18nContext = createContext<QueryPlanTranslate | undefined>(
  undefined
);

export function QueryPlanI18nProvider({
  translate,
  children,
}: {
  translate?: QueryPlanTranslate;
  children: ReactNode;
}) {
  return (
    <QueryPlanI18nContext.Provider value={translate}>
      {children}
    </QueryPlanI18nContext.Provider>
  );
}

export function useQueryPlanTranslation(): QueryPlanTranslate {
  return useContext(QueryPlanI18nContext) ?? fallbackTranslate;
}

/**
 * The words a warning is shown in. This viewer's own advice is read in the
 * reader's language; a warning the engine reported keeps the engine's words,
 * as the plan's own operator names and values do.
 */
export function planWarningText(
  warning: PlanWarning,
  translate: QueryPlanTranslate
): { title: string; detail: string } {
  if (!warning.key) {
    return { title: warning.title ?? "", detail: warning.detail ?? "" };
  }
  return {
    title: translate(`warning.${warning.key}.title`),
    detail: translate(`warning.${warning.key}.detail`),
  };
}
