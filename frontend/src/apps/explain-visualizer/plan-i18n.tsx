import { createContext, type ReactNode, useContext } from "react";

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
  "error.postgres-empty":
    "PostgreSQL returned no query plan for this statement.",
  "error.postgres-invalid":
    "The query plan is not valid JSON. EXPLAIN (FORMAT JSON) output is expected.",
  "error.postgres-no-plan":
    'The query plan JSON does not contain a "Plan" object.',
  "error.spanner-emulator":
    "Spanner returned no query plan for this statement. The Spanner emulator returns none for any statement, so explain it on a Spanner instance to see one.",
  "error.spanner-empty": "Spanner returned no query plan for this statement.",
  "error.spanner-invalid":
    "The query plan is not valid JSON. A Spanner query plan is expected.",
  "error.spanner-no-plan":
    'The query plan JSON does not contain a "planNodes" list with an operator at its root.',
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
