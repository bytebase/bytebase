import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { colorizeStatement } from "@/components/monaco/core";
import { useInViewOnce } from "@/hooks/useInViewOnce";
import { useSheetStatement } from "@/hooks/useSheetStatement";
import { cn } from "@/lib/utils";
import type { StatementAnchor } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  derivePlanChangeReference,
  type PlanChangeReferenceResources,
} from "../../utils/changeReference";
import { PlanChangeReference } from "../PlanChangeReference";
import type { ThreadBarProps } from "./CommentThreadCard";
import { tokenizeLines } from "./placement/lineTokens";
import { anchorLineRange, type Placement } from "./placement/place";
import {
  type AnchorState,
  completePreviewLineCount,
  excerptLines,
  formatLineRange,
  type LineRange,
  resolveAnchorState,
  sheetNameOfSha256,
  targetSha256OfSpec,
} from "./threadModel";

// What an Activity card shows about where its thread is: the bar's title,
// lines, placement state and link, and the SQL excerpt of the recorded sheet
// revision for the body. The wrapper is observed so the recorded revision is
// fetched only once the card nears the viewport, folded or not. The editor
// never renders this; there the anchor is a gutter marker.
export function StatementAnchorContext({
  anchor,
  changeReferenceResources,
  children,
  onViewInStatement,
  placement,
  plan,
  project,
}: {
  anchor: StatementAnchor | undefined;
  changeReferenceResources: PlanChangeReferenceResources;
  children: (anchored: {
    bar: ThreadBarProps;
    excerpt?: ReactNode;
  }) => ReactNode;
  onViewInStatement?: () => void;
  placement: Placement | undefined;
  plan: Plan;
  project: Project | undefined;
}) {
  const { t } = useTranslation();
  const range = anchor ? anchorLineRange(anchor) : undefined;
  const specIndex = plan.specs.findIndex(
    (candidate) => candidate.id === anchor?.spec
  );
  const spec = specIndex < 0 ? undefined : plan.specs[specIndex];
  const sheetName =
    project && anchor
      ? sheetNameOfSha256(project.name, anchor.sheetSha256)
      : "";
  const { ref, inView } = useInViewOnce<HTMLDivElement>();
  const enabled = sheetName !== "" && inView;
  const { statement, isLoading, isTruncated } = useSheetStatement({
    enabled,
    sheetName,
  });
  const loaded = enabled && !isLoading;
  const targetSha256 = targetSha256OfSpec(spec);
  const previewEndLine = useMemo(
    () => completePreviewLineCount(statement, isTruncated),
    [isTruncated, statement]
  );
  const recordedLinesVisible = Boolean(
    statement && range && range.endLine <= previewEndLine
  );
  let state: AnchorState | undefined;
  if (anchor) {
    const resolved = resolveAnchorState(anchor, plan, placement);
    const onCurrentSheet = anchor.sheetSha256 === targetSha256;
    state = resolved;
    // A hash match is current only when its complete lines are in the preview.
    if (resolved === "CURRENT" && onCurrentSheet) {
      if (!loaded) state = "PENDING";
      else if (!recordedLinesVisible) state = "UNAVAILABLE";
    }
    // A prior revision with visible original lines remains discoverable as
    // Outdated when a comparison cannot establish a current position.
    if (
      resolved === "UNAVAILABLE" &&
      project &&
      targetSha256 &&
      !onCurrentSheet
    ) {
      if (!loaded) state = "PENDING";
      else if (recordedLinesVisible) state = "OUTDATED";
    }
  }
  const stateTooltip =
    state === "OUTDATED" &&
    placement?.state === "UNAVAILABLE" &&
    placement.reason === "SIZE_LIMIT"
      ? t("plan.review.thread.anchor.comparison-size-limit")
      : undefined;

  const reference = spec
    ? derivePlanChangeReference({
        index: specIndex,
        resources: changeReferenceResources,
        siblings: plan.specs,
        spec,
        t,
      })
    : undefined;
  const titleText = !anchor
    ? t("plan.review.thread.comment-on-plan")
    : (reference?.label ?? t("plan.review.thread.anchor.unavailable-change"));
  const bar: ThreadBarProps = {
    title: reference ? (
      <PlanChangeReference reference={reference} tone="control" />
    ) : (
      titleText
    ),
    foldLabel: range
      ? t("plan.review.thread.fold-label", {
          title: titleText,
          count: range.endLine - range.startLine + 1,
          range: formatLineRange(range),
        })
      : titleText,
    lines: range,
    anchorState: state,
    stateTooltip,
    onViewInStatement: state === "CURRENT" ? onViewInStatement : undefined,
  };
  const excerpt =
    !isLoading && statement && range ? (
      <SqlExcerpt
        sheetName={sheetName}
        dimmed={state === "OUTDATED" || state === "UNAVAILABLE"}
        statement={statement}
        range={range}
      />
    ) : undefined;

  return (
    <div ref={ref} className="min-w-0">
      {children({ bar, excerpt })}
    </div>
  );
}

const CONTEXT_LINES_BEFORE = 2;

// Sheets are immutable and content-addressed, so each is tokenized once per
// page however many cards excerpt it, and only through the deepest line a
// card needs rather than the whole sheet. Tokenizing from the document start
// keeps a snippet that begins inside a multiline comment or string in the
// same lexical state as the editor. A few sheets are kept; a plan rarely
// excerpts more revisions than that at once.
const COLORIZED_SHEET_LIMIT = 8;
const colorizedSheets = new Map<
  string,
  { throughLine: number; lines: Promise<string[]> }
>();
const colorizedLinesOf = (
  sheetName: string,
  statement: string,
  throughLine: number
): Promise<string[]> => {
  const cached = colorizedSheets.get(sheetName);
  if (cached && cached.throughLine >= throughLine) return cached.lines;
  const prefix = tokenizeLines(statement).slice(0, throughLine).join("");
  const lines = colorizeStatement(prefix).then((html) =>
    html.split(/<br\s*\/?>/)
  );
  colorizedSheets.delete(sheetName);
  colorizedSheets.set(sheetName, { throughLine, lines });
  for (const key of colorizedSheets.keys()) {
    if (colorizedSheets.size <= COLORIZED_SHEET_LIMIT) break;
    colorizedSheets.delete(key);
  }
  return lines;
};

function SqlExcerpt({
  dimmed,
  sheetName,
  statement,
  range,
}: {
  dimmed: boolean;
  sheetName: string;
  statement: string;
  range: LineRange;
}) {
  const startLine = Math.max(1, range.startLine - CONTEXT_LINES_BEFORE);
  const lines = useMemo(
    () => excerptLines(statement, { startLine, endLine: range.endLine }),
    [range.endLine, startLine, statement]
  );
  const [colorized, setColorized] = useState<{
    sheetName: string;
    lines: string[];
  }>();
  useEffect(() => {
    let cancelled = false;
    void colorizedLinesOf(sheetName, statement, range.endLine).then((html) => {
      if (!cancelled) setColorized({ sheetName, lines: html });
    });
    return () => {
      cancelled = true;
    };
  }, [range.endLine, sheetName, statement]);
  const htmlLines =
    colorized?.sheetName === sheetName ? colorized.lines : undefined;
  if (lines.length === 0) return null;
  return (
    <pre
      className={cn(
        "m-0 overflow-x-auto bg-background font-mono text-sm leading-6",
        dimmed && "opacity-60"
      )}
    >
      <div className="w-max min-w-full">
        {lines.map((line, index) => {
          const lineNumber = startLine + index;
          const selected =
            lineNumber >= range.startLine && lineNumber <= range.endLine;
          const html = htmlLines?.[lineNumber - 1];
          return (
            <div
              className={cn(
                "flex items-start gap-x-3 px-3",
                selected && "bg-accent/12"
              )}
              data-line-number={lineNumber}
              data-selected={selected || undefined}
              key={lineNumber}
            >
              <span
                className="shrink-0 select-none text-right text-control-placeholder"
                style={{
                  width: `${Math.max(2, String(range.endLine).length)}ch`,
                }}
              >
                {lineNumber}
              </span>
              {html === undefined ? (
                <code className="min-w-0 flex-1 whitespace-pre text-main">
                  {line}
                </code>
              ) : (
                <code
                  className="min-w-0 flex-1 whitespace-pre text-main"
                  dangerouslySetInnerHTML={{ __html: html }}
                />
              )}
            </div>
          );
        })}
      </div>
    </pre>
  );
}
