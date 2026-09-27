import { createContext, type ReactNode, useContext } from "react";

const ENGLISH = {
  "copy.copied": "Copied",
  "copy.failed": "Copy failed",
  "copy.plan": "Copy plan",
  "copy.query": "Copy query",
  "details.select-node": "Select a node to see its details.",
  "diagram.collapse": "Collapse {{count}} nodes under {{node}}",
  "diagram.cost": "cost {{cost}}",
  "diagram.estimated-rows": "{{count}} estimated rows",
  "diagram.expand": "Expand {{count}} nodes under {{node}}",
  "diagram.fit": "Fit to view",
  "diagram.nodes-hidden": "{{count}} nodes hidden",
  "diagram.plan-cost-share": "{{share}} of plan cost",
  "diagram.rows": "rows {{count}}",
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
  "error.unreadable": "This query plan could not be read",
  "grid.added-cost": "Added cost",
  "grid.added-cost-hint":
    "Cost this node adds on top of its children; some engines also count an input's repeats here, as under a nested loop. The bar is that cost as a share of the plan.",
  "grid.caption":
    "Plan nodes in the order the planner nests them, parents before children.",
  "grid.estimated-rows": "Estimated rows",
  "grid.node": "Node",
  "grid.rows": "Rows",
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
  "legend.cost":
    "each card's bar is that node's share of the plan's estimated cost",
  "legend.rows":
    "an edge thickens with the rows its child is estimated to return",
  "metric.added-cost": "Added cost",
  "metric.bytes": "{{count}} bytes",
  "metric.estimated-rows": "Estimated rows",
  "metric.row-width": "Row width",
  "metric.startup-cost": "Startup cost",
  "metric.total-cost": "Total cost",
  "query.missing": "The statement was not captured with this plan.",
  "summary.added-cost": "Added cost",
  "summary.cost-by-operation": "Cost by operation",
  "summary.cost-by-operation-description":
    "Every node type in the plan and what it costs in total.",
  "summary.cost-ranges": "Cost ranges",
  "summary.cost-ranges-description":
    "Each node spans its startup cost — the cost before its first row — to its total cost, both of which include everything below it. Darker spans carry more of the plan's cost. Costs are not a schedule: two inputs of a join overlap here but run one after the other.",
  "summary.costliest": "Costliest operators",
  "summary.costliest-description":
    "Ranked by the cost a node adds on top of its children. Where an input runs repeatedly, as under a nested loop, some engines count the repeats in the node that reruns it rather than in the input itself.",
  "summary.count": "Count",
  "summary.estimate-note":
    "The statement was planned but not run, so every figure here is an optimizer estimate rather than a measurement. Cost units are arbitrary and comparable only within this plan.",
  "summary.estimated-rows-returned": "Estimated rows returned",
  "summary.no-added-cost":
    "No node in this plan adds an estimated cost of its own.",
  "summary.nodes": "Nodes",
  "summary.operation": "Operation",
  "summary.share": "Share",
  "summary.timeline-node":
    "{{name}}, cost {{startup}} to {{total}}, {{share}} of plan cost",
  "summary.total-estimated-cost": "Total estimated cost",
  "tab.diagram": "Diagram",
  "tab.grid": "Grid",
  "tab.query": "Query",
  "tab.summary": "Summary",
  "tab.text": "Text",
  "totals.estimated-cost": "estimated cost {{cost}}",
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
