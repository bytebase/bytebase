import { Info } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useMemo, useState } from "react";
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
  type QueryPlanTranslate,
  type QueryPlanTranslationKey,
  useQueryPlanTranslation,
} from "./plan-i18n";
import {
  findPlanNode,
  formatPlanCost,
  formatPlanCount,
  type PlanEstimates,
  type PlanHighlightMode,
  type PlanNode,
  type PlanTree,
  planCollapsedAncestor,
  planCostliestNodes,
  planRevealNode,
} from "./plan-model";
import { QueryPlanDiagram } from "./QueryPlanDiagram";
import { QueryPlanGrid } from "./QueryPlanGrid";
import { QueryPlanNodeDetails } from "./QueryPlanNodeDetails";
import { QueryPlanSummary } from "./QueryPlanSummary";

interface Props {
  readonly tree: PlanTree;
  /** The machine-readable plan used to build the viewer. */
  readonly rawPlan: string;
  /** The plan output shown on the text tab, when it differs from rawPlan. */
  readonly textPlan?: string;
  /** Shown at the end of the tab row, such as a reminder that the plan is an estimate. */
  readonly note?: ReactNode;
  readonly disallowCopyingData?: boolean;
}

type TabValue = "diagram" | "grid" | "text";

interface HighlightOption {
  readonly value: PlanHighlightMode;
  readonly label: QueryPlanTranslationKey;
  readonly hint: QueryPlanTranslationKey;
  /** Whether a plan carries the estimate the option shades by. */
  readonly available: (estimates: PlanEstimates) => boolean;
}

const HIGHLIGHT_OPTIONS: readonly HighlightOption[] = [
  {
    value: "off",
    label: "highlight.off",
    hint: "highlight.off-hint",
    available: () => true,
  },
  {
    value: "cost",
    label: "highlight.cost",
    // Total cost is cumulative, so shading by it would always make the root
    // darkest. Self cost is what the node adds on top of its children.
    hint: "highlight.cost-hint",
    available: (estimates) => estimates.cost,
  },
  {
    value: "rows",
    label: "highlight.rows",
    hint: "highlight.rows-hint",
    available: (estimates) => estimates.rows,
  },
];

const RESIZE_HANDLE_BASE_CLASS =
  "shrink-0 bg-control-border transition-colors hover:bg-accent data-[resize-handle-active]:bg-accent";

/** Below this width the diagram and its details stack instead of splitting. */
const STACK_QUERY = "(max-width: 767px)";

/** One button per shading mode the plan's estimates support. */
function HighlightControl({
  options,
  value,
  onChange,
}: {
  options: readonly HighlightOption[];
  value: PlanHighlightMode;
  onChange: (next: PlanHighlightMode) => void;
}) {
  const translate = useQueryPlanTranslation();
  return (
    <div
      role="group"
      aria-label={translate("highlight.label")}
      className="inline-flex items-center gap-px overflow-hidden rounded-xs border border-control-border bg-control-border"
    >
      {options.map((option) => {
        const selected = option.value === value;
        return (
          <Tooltip key={option.value} content={translate(option.hint)}>
            <Button
              size="sm"
              appearance={selected ? "solid" : "secondary"}
              aria-pressed={selected}
              className={cn("rounded-none", !selected && "bg-background")}
              onClick={() => onChange(option.value)}
            >
              {translate(option.label)}
            </Button>
          </Tooltip>
        );
      })}
    </div>
  );
}

/** The plan's size and cost; `legend`, when given, explains the diagram on hover. */
function PlanTotalsLine({ tree, legend }: { tree: PlanTree; legend?: string }) {
  const translate = useQueryPlanTranslation();
  return (
    <p className="flex items-center gap-1 text-xs leading-4 text-control-light">
      {[
        translate("totals.nodes", {
          count: formatPlanCount(tree.nodes.length),
        }),
        tree.estimates.cost
          ? translate("totals.estimated-cost", {
              cost: formatPlanCost(tree.root.totalCost),
            })
          : undefined,
      ]
        .filter(Boolean)
        .join(" · ")}
      {legend ? (
        <Tooltip content={legend}>
          <Info aria-label={legend} className="size-3.5 shrink-0" />
        </Tooltip>
      ) : null}
    </p>
  );
}

/** What the diagram's bars and edge widths encode, for the estimates it has. */
function diagramLegend(
  estimates: PlanEstimates,
  translate: QueryPlanTranslate
): string | undefined {
  const parts = [
    estimates.cost ? translate("legend.cost") : undefined,
    estimates.rows ? translate("legend.rows") : undefined,
  ].filter(Boolean);
  if (parts.length === 0) return undefined;
  const sentence = `${parts.join("; ")}.`;
  return sentence.charAt(0).toUpperCase() + sentence.slice(1);
}

/**
 * Puts one plan representation beside the side panel: the ranked costliest
 * operators, when the plan has costs, above the selected node's details. The
 * diagram and the grid are two readings of the same selection, so they share
 * this frame instead of each carrying its own copy.
 */
function PlanWithDetails({
  stacked,
  node,
  summary,
  children,
}: {
  stacked: boolean;
  node: PlanNode | undefined;
  summary: ReactNode;
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
        <div className="flex h-full min-h-0 flex-col gap-4 overflow-auto p-4">
          {summary}
          <QueryPlanNodeDetails node={node} />
        </div>
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
export function formatPlanSource(source: string): string {
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
  disallowCopyingData,
  children,
}: {
  content: string;
  label: string;
  disallowCopyingData: boolean;
  children: ReactNode;
}) {
  return (
    <>
      {disallowCopyingData ? null : (
        <div className="flex shrink-0 items-center justify-end px-4 py-2">
          <PlanCopyButton content={content} label={label} />
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-auto px-4 pb-4">{children}</div>
    </>
  );
}

const MONOSPACE_BLOCK_CLASS =
  "font-mono text-xs leading-4 break-words whitespace-pre-wrap text-main";

/**
 * The text plan tab's content. The tab mounts it only while open, so a large
 * plan is formatted when someone reads it rather than on every page load.
 */
function TextPlan({
  source,
  disallowCopyingData,
}: {
  source: string;
  disallowCopyingData: boolean;
}) {
  const translate = useQueryPlanTranslation();
  const formatted = useMemo(() => formatPlanSource(source), [source]);
  return (
    <CopyablePanel
      content={formatted}
      label={translate("copy.plan")}
      disallowCopyingData={disallowCopyingData}
    >
      <pre className={MONOSPACE_BLOCK_CLASS}>{formatted}</pre>
    </CopyablePanel>
  );
}

export function QueryPlanViewer({
  tree,
  rawPlan,
  textPlan = rawPlan,
  note,
  disallowCopyingData = false,
}: Props) {
  const translate = useQueryPlanTranslation();
  const [tab, setTab] = useState<TabValue>("diagram");
  // The diagram opens centered on the costliest operator, so the details beside
  // it start there too.
  const [selectedId, setSelectedId] = useState<string>(
    () => planCostliestNodes(tree, 1)[0]?.id ?? tree.root.id
  );
  const [highlight, setHighlight] = useState<PlanHighlightMode>("off");
  // Collapsed subtrees live here rather than in the diagram so switching tabs
  // does not fold the plan back open.
  const [collapsedIds, setCollapsedIds] = useState<ReadonlySet<string>>(
    () => new Set()
  );
  // A node the diagram should scroll to, rather than merely mark as selected.
  const [revealId, setRevealId] = useState<string | undefined>();

  const selectedNode = findPlanNode(tree, selectedId);
  const stacked = useMediaQuery(STACK_QUERY);
  const highlightOptions = HIGHLIGHT_OPTIONS.filter((option) =>
    option.available(tree.estimates)
  );
  const legend = diagramLegend(tree.estimates, translate);

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

  const onRevealed = useCallback(() => setRevealId(undefined), []);

  // Choosing a ranked operator also brings it into view on the diagram.
  const summary = tree.estimates.cost ? (
    <QueryPlanSummary
      tree={tree}
      selectedId={selectedId}
      onSelect={(id) => {
        selectNode(id);
        if (tab === "diagram") setRevealId(id);
      }}
    />
  ) : null;

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => setTab(value as TabValue)}
      className={cn(
        "flex h-full min-h-0 w-full flex-col bg-background text-main",
        disallowCopyingData && "select-none"
      )}
    >
      <TabsList className="shrink-0 flex-wrap px-4 pt-3">
        <TabsTrigger value="diagram">{translate("tab.diagram")}</TabsTrigger>
        <TabsTrigger value="grid">{translate("tab.grid")}</TabsTrigger>
        <TabsTrigger value="text">{translate("tab.text")}</TabsTrigger>
        {note ? (
          <span className="ml-auto self-center text-sm text-control-light">
            {note}
          </span>
        ) : null}
      </TabsList>

      <TabsPanel
        value="diagram"
        // The diagram holds the reader's zoom and pan, which unmounting would
        // throw away; the other tabs keep nothing a remount would lose.
        keepMounted
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <PlanWithDetails
          stacked={stacked}
          node={selectedNode}
          summary={summary}
        >
          <div
            className={cn(
              "flex shrink-0 flex-wrap items-center gap-4 px-4 py-2",
              highlightOptions.length > 1 ? "justify-between" : "justify-end"
            )}
          >
            {highlightOptions.length > 1 ? (
              <div className="flex items-center gap-2">
                <span className="text-xs leading-4 text-control-light">
                  {translate("highlight.label")}
                </span>
                <HighlightControl
                  options={highlightOptions}
                  value={highlight}
                  onChange={setHighlight}
                />
              </div>
            ) : null}
            <PlanTotalsLine tree={tree} legend={legend} />
          </div>
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
        <PlanWithDetails
          stacked={stacked}
          node={selectedNode}
          summary={summary}
        >
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

      <TabsPanel
        value="text"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <TextPlan source={textPlan} disallowCopyingData={disallowCopyingData} />
      </TabsPanel>
    </Tabs>
  );
}
