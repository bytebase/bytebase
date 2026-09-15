import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip } from "@/components/ui/tooltip";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { PlanCopyButton } from "./PlanCopyButton";
import {
  findPlanNode,
  formatPlanCost,
  formatPlanCount,
  type PlanEstimates,
  type PlanHighlightMode,
  type PlanNode,
  type PlanTree,
  planCollapsedAncestor,
  planNodeFragment,
  planNodeIdFromFragment,
  planRevealNode,
} from "./plan-model";
import { QueryPlanDiagram } from "./QueryPlanDiagram";
import { QueryPlanGrid } from "./QueryPlanGrid";
import { QueryPlanNodeDetails } from "./QueryPlanNodeDetails";
import { QueryPlanSummary } from "./QueryPlanSummary";

interface Props {
  readonly tree: PlanTree;
  /** The engine's plan output, shown on the raw tab. */
  readonly rawPlan: string;
  readonly query?: string;
}

type TabValue = "diagram" | "grid" | "summary" | "raw" | "query";

interface HighlightOption {
  readonly value: PlanHighlightMode;
  readonly label: string;
  readonly hint: string;
  /** Whether a plan carries the estimate the option shades by. */
  readonly available: (estimates: PlanEstimates) => boolean;
}

const HIGHLIGHT_OPTIONS: readonly HighlightOption[] = [
  {
    value: "off",
    label: "Off",
    hint: "Leave every node card unshaded",
    available: () => true,
  },
  {
    value: "cost",
    label: "Cost",
    // Total cost is cumulative, so shading by it would always make the root
    // darkest. Self cost is what the node adds on top of its children.
    hint: "Shade by the cost a node adds on top of its children",
    available: (estimates) => estimates.cost,
  },
  {
    value: "rows",
    label: "Rows",
    hint: "Shade by the estimated row count",
    available: (estimates) => estimates.rows,
  },
];

const RESIZE_HANDLE_BASE_CLASS =
  "shrink-0 bg-control-border transition-colors hover:bg-accent data-[resize-handle-active]:bg-accent";

/** Below this width the diagram and its details stack instead of splitting. */
const STACK_QUERY = "(max-width: 767px)";

/**
 * A toggle group rather than `SegmentedControl`: this entry never loads the
 * main app's stylesheet, which is where the build injects StyleX's rules, so a
 * StyleX-sized control would render unstyled here.
 */
function HighlightControl({
  options,
  value,
  onChange,
}: {
  options: readonly HighlightOption[];
  value: PlanHighlightMode;
  onChange: (next: PlanHighlightMode) => void;
}) {
  return (
    <div
      role="group"
      aria-label="Highlight nodes by"
      className="inline-flex items-center gap-px overflow-hidden rounded-xs border border-control-border bg-control-border"
    >
      {options.map((option) => {
        const selected = option.value === value;
        return (
          <Tooltip key={option.value} content={option.hint}>
            <Button
              size="sm"
              appearance={selected ? "solid" : "secondary"}
              aria-pressed={selected}
              className={cn("rounded-none", !selected && "bg-background")}
              onClick={() => onChange(option.value)}
            >
              {option.label}
            </Button>
          </Tooltip>
        );
      })}
    </div>
  );
}

function PlanTotalsLine({ tree }: { tree: PlanTree }) {
  return (
    <p className="text-xs leading-4 text-control-light">
      {[
        `${formatPlanCount(tree.nodes.length)} nodes`,
        tree.estimates.cost
          ? `estimated cost ${formatPlanCost(tree.root.totalCost)}`
          : undefined,
      ]
        .filter(Boolean)
        .join(" · ")}
    </p>
  );
}

/** What the diagram's bars and edge widths encode, for the estimates it has. */
function diagramLegend(estimates: PlanEstimates): string | undefined {
  const parts = [
    estimates.cost
      ? "each card's bar is that node's share of the plan's estimated cost"
      : undefined,
    estimates.rows
      ? "an edge thickens with the rows its child is estimated to return"
      : undefined,
  ].filter(Boolean);
  if (parts.length === 0) return undefined;
  const sentence = `${parts.join("; ")}.`;
  return sentence.charAt(0).toUpperCase() + sentence.slice(1);
}

/**
 * Puts one plan representation beside the detail pane. The diagram and the grid
 * are two readings of the same selection, so they share this frame instead of
 * each carrying its own copy of the details.
 */
function PlanWithDetails({
  stacked,
  node,
  children,
}: {
  stacked: boolean;
  node: PlanNode | undefined;
  children: ReactNode;
}) {
  return (
    <PanelGroup
      orientation={stacked ? "vertical" : "horizontal"}
      className="h-full min-h-0 w-full"
    >
      <Panel defaultSize={stacked ? "60%" : "70%"} minSize="30%">
        <div className="flex h-full min-h-0 flex-col">{children}</div>
      </Panel>
      <PanelResizeHandle
        className={cn(
          RESIZE_HANDLE_BASE_CLASS,
          stacked ? "h-1 w-full cursor-ns-resize" : "w-1 cursor-ew-resize"
        )}
      />
      <Panel defaultSize={stacked ? "40%" : "30%"} minSize="20%">
        <QueryPlanNodeDetails node={node} />
      </Panel>
    </PanelGroup>
  );
}

const XML_INDENT = "  ";

function escapeXmlText(value: string): string {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;");
}

function escapeXmlAttribute(value: string): string {
  // A literal line break inside a value is read back as a space, so breaks
  // stay escaped and a copied plan keeps the engine's values.
  return escapeXmlText(value)
    .replaceAll('"', "&quot;")
    .replaceAll("\n", "&#10;")
    .replaceAll("\r", "&#13;")
    .replaceAll("\t", "&#9;");
}

/**
 * An XML document with one element per line, or undefined when it does not
 * parse. Plans are elements and attributes only, so text is kept solely in
 * elements that have no children.
 */
function indentXml(source: string): string | undefined {
  const document = new DOMParser().parseFromString(source, "application/xml");
  if (document.getElementsByTagName("parsererror").length > 0) return undefined;
  const lines: string[] = [];
  const visit = (element: Element, depth: number) => {
    const indent = XML_INDENT.repeat(depth);
    const attributes = Array.from(element.attributes)
      .map(
        (attribute) =>
          ` ${attribute.name}="${escapeXmlAttribute(attribute.value)}"`
      )
      .join("");
    const tag = element.tagName;
    if (element.children.length === 0) {
      const text = element.textContent ?? "";
      lines.push(
        text
          ? `${indent}<${tag}${attributes}>${escapeXmlText(text)}</${tag}>`
          : `${indent}<${tag}${attributes}/>`
      );
      return;
    }
    lines.push(`${indent}<${tag}${attributes}>`);
    for (const child of Array.from(element.children)) visit(child, depth + 1);
    lines.push(`${indent}</${tag}>`);
  };
  visit(document.documentElement, 0);
  return lines.join("\n");
}

/** The plan indented for reading when it is JSON or XML, else as it came. */
function formatPlanSource(source: string): string {
  if (source.trimStart().startsWith("<")) {
    return indentXml(source.trim()) ?? source;
  }
  try {
    return JSON.stringify(JSON.parse(source), null, 2);
  } catch {
    return source;
  }
}

/** A panel whose content scrolls under a toolbar holding its copy control. */
function CopyablePanel({
  content,
  label,
  children,
}: {
  content: string;
  label: string;
  children: ReactNode;
}) {
  return (
    <>
      <div className="flex shrink-0 items-center justify-end px-4 py-2">
        <PlanCopyButton content={content} label={label} />
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-4 pb-4">{children}</div>
    </>
  );
}

const MONOSPACE_BLOCK_CLASS =
  "font-mono text-xs leading-4 break-words whitespace-pre-wrap text-main";

/**
 * The raw plan tab's content. The tab mounts it only while open, so a large
 * plan is formatted when someone reads it rather than on every page load.
 */
function RawPlan({ source }: { source: string }) {
  const formatted = useMemo(() => formatPlanSource(source), [source]);
  return (
    <CopyablePanel content={formatted} label="Copy plan">
      <pre className={MONOSPACE_BLOCK_CLASS}>{formatted}</pre>
    </CopyablePanel>
  );
}

/**
 * The node a fragment names, ignoring one that names a node this plan does not
 * have: a stale or hand-edited link should open the plan, not break it.
 */
function fragmentSelection(tree: PlanTree): string | undefined {
  const id = planNodeIdFromFragment(location.hash);
  return findPlanNode(tree, id)?.id;
}

export function QueryPlanViewer({ tree, rawPlan, query }: Props) {
  const [tab, setTab] = useState<TabValue>("diagram");
  const [selectedId, setSelectedId] = useState<string>(
    () => fragmentSelection(tree) ?? tree.root.id
  );
  const [highlight, setHighlight] = useState<PlanHighlightMode>("off");
  // Collapsed subtrees live here rather than in the diagram so switching tabs
  // does not fold the plan back open.
  const [collapsedIds, setCollapsedIds] = useState<ReadonlySet<string>>(
    () => new Set()
  );
  // A node the diagram should scroll to, rather than merely mark as selected.
  const [revealId, setRevealId] = useState<string | undefined>(() =>
    fragmentSelection(tree)
  );

  const selectedNode = findPlanNode(tree, selectedId);
  const stacked = useMediaQuery(STACK_QUERY);
  const highlightOptions = HIGHLIGHT_OPTIONS.filter((option) =>
    option.available(tree.estimates)
  );
  const legend = diagramLegend(tree.estimates);

  const selectNode = useCallback(
    (id: string) => {
      setSelectedId(id);
      // The grid and the summary list nodes the diagram may have folded away.
      // Opening the collapse that hides one keeps every surface agreeing on
      // where the selection is.
      setCollapsedIds((previous) => planRevealNode(tree.root, previous, id));
    },
    [tree]
  );

  const toggleCollapse = useCallback(
    (id: string) => {
      const next = new Set(collapsedIds);
      if (!next.delete(id)) next.add(id);
      setCollapsedIds(next);
      // Folding a subtree away must not leave the selection inside it with no
      // card on screen; the collapse that swallowed it takes the selection.
      const hiding = planCollapsedAncestor(tree.root, next, selectedId);
      if (hiding) setSelectedId(hiding.id);
    },
    [collapsedIds, selectedId, tree]
  );

  // Naming the selection in the fragment is what makes a node shareable. It
  // replaces rather than pushes, so Back leaves the page instead of walking
  // through every node the reader clicked on the way here.
  //
  // Only a selection the reader moved is written: naming the root on load
  // would turn every reload into a followed deep link, which reveals the node
  // and takes the view away from the fit the page opened on.
  const writtenSelection = useRef(selectedId);
  useEffect(() => {
    if (selectedId === writtenSelection.current) return;
    writtenSelection.current = selectedId;
    const fragment = planNodeFragment(selectedId);
    if (location.hash !== fragment) {
      history.replaceState(null, "", fragment);
    }
  }, [selectedId]);

  // Following a second link to the page it is already on changes the fragment
  // without reloading anything.
  useEffect(() => {
    const onHashChange = () => {
      const id = fragmentSelection(tree);
      if (id === undefined) return;
      selectNode(id);
      setRevealId(id);
    };
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, [tree, selectNode]);

  const onRevealed = useCallback(() => setRevealId(undefined), []);

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => setTab(value as TabValue)}
      className="flex h-full min-h-0 w-full flex-col bg-background text-main"
    >
      <TabsList className="shrink-0 flex-wrap px-4 pt-3">
        <TabsTrigger value="diagram">Diagram</TabsTrigger>
        <TabsTrigger value="grid">Grid</TabsTrigger>
        {/* The summary is where the plan's cost goes, which a plan without
            cost estimates cannot say. */}
        {tree.estimates.cost ? (
          <TabsTrigger value="summary">Summary</TabsTrigger>
        ) : null}
        <TabsTrigger value="raw">Raw plan</TabsTrigger>
        <TabsTrigger value="query">Query</TabsTrigger>
      </TabsList>

      <TabsPanel
        value="diagram"
        // The diagram holds the reader's zoom and pan, which unmounting would
        // throw away; the other tabs keep nothing a remount would lose.
        keepMounted
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <PlanWithDetails stacked={stacked} node={selectedNode}>
          <div
            className={cn(
              "flex shrink-0 flex-wrap items-center gap-4 px-4 py-2",
              highlightOptions.length > 1 ? "justify-between" : "justify-end"
            )}
          >
            {highlightOptions.length > 1 ? (
              <div className="flex items-center gap-2">
                <span className="text-xs leading-4 text-control-light">
                  Highlight
                </span>
                <HighlightControl
                  options={highlightOptions}
                  value={highlight}
                  onChange={setHighlight}
                />
              </div>
            ) : null}
            <PlanTotalsLine tree={tree} />
          </div>
          {legend ? (
            <p className="shrink-0 px-4 pb-2 text-xs leading-4 text-control-light">
              {legend}
            </p>
          ) : null}
          <QueryPlanDiagram
            tree={tree}
            selectedId={selectedId}
            onSelect={selectNode}
            highlight={highlight}
            collapsedIds={collapsedIds}
            onToggleCollapse={toggleCollapse}
            revealId={revealId}
            onRevealed={onRevealed}
          />
        </PlanWithDetails>
      </TabsPanel>

      <TabsPanel
        value="grid"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <PlanWithDetails stacked={stacked} node={selectedNode}>
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-4 px-4 py-2">
            <PlanTotalsLine tree={tree} />
          </div>
          <QueryPlanGrid
            tree={tree}
            selectedId={selectedId}
            onSelect={selectNode}
          />
        </PlanWithDetails>
      </TabsPanel>

      {tree.estimates.cost ? (
        <TabsPanel
          value="summary"
          className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <PlanWithDetails stacked={stacked} node={selectedNode}>
            <QueryPlanSummary
              tree={tree}
              selectedId={selectedId}
              onSelect={selectNode}
            />
          </PlanWithDetails>
        </TabsPanel>
      ) : null}

      <TabsPanel
        value="raw"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <RawPlan source={rawPlan} />
      </TabsPanel>

      <TabsPanel
        value="query"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        {query ? (
          <CopyablePanel content={query} label="Copy query">
            <pre className={MONOSPACE_BLOCK_CLASS}>{query}</pre>
          </CopyablePanel>
        ) : (
          <p className="p-4 text-sm leading-5 text-control-light">
            The statement was not captured with this plan.
          </p>
        )}
      </TabsPanel>
    </Tabs>
  );
}
