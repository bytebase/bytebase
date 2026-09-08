import { Ban, ExternalLink, History } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { PlanChangeReferenceRenderer } from "@/components/issue-activity/IssueCommentActivity";
import { colorizeStatement } from "@/components/monaco/core";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useSheetStatement } from "@/hooks/useSheetStatement";
import { cn } from "@/lib/utils";
import type { StatementAnchor } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { tokenizeLines } from "./placement/lineTokens";
import type { Placement } from "./placement/place";
import {
  type AnchorState,
  anchorLineRange,
  excerptLines,
  type LineRange,
  lineRangeLabel,
  resolveAnchorState,
  sheetNameOfSha256,
} from "./threadModel";

// The recorded context of an anchored comment in the Review Activity
// timeline: the change, the originally selected lines, the placement state,
// and the SQL excerpt of the recorded sheet revision. The editor never
// renders this block; there the anchor is a gutter marker and a line
// highlight at the comment's current placement.
export function StatementAnchorContext({
  anchor,
  onViewInStatement,
  placement,
  plan,
  project,
  renderPlanChangeReference,
}: {
  anchor: StatementAnchor;
  onViewInStatement?: () => void;
  placement: Placement | undefined;
  plan: Plan;
  project: Project | undefined;
  renderPlanChangeReference: PlanChangeReferenceRenderer;
}) {
  const { t } = useTranslation();
  const state = resolveAnchorState(anchor, plan, placement);
  const range = anchorLineRange(anchor);
  const spec = plan.specs.find((candidate) => candidate.id === anchor.spec);
  const sheetName = project
    ? sheetNameOfSha256(project.name, anchor.sheetSha256)
    : "";
  const { statement, isLoading } = useSheetStatement({
    enabled: sheetName !== "",
    sheetName,
  });

  return (
    <div
      className="@container/anchor flex min-w-0 flex-col"
      data-anchor-state={state}
      data-testid="statement-anchor"
    >
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5 border-b border-block-border bg-control-bg/50 px-3 py-1.5 text-xs">
        <div className="flex min-w-0 flex-1 basis-full @sm/anchor:basis-0">
          {spec ? (
            renderPlanChangeReference({ siblings: plan.specs, spec })
          ) : (
            <span className="truncate font-medium text-control-light">
              {t("plan.review.thread.anchor.unavailable-change")}
            </span>
          )}
        </div>
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5">
          {range && (
            <span className="shrink-0 text-control-light">
              {lineRangeLabel(t, range)}
            </span>
          )}
          <AnchorStatePill
            onViewInStatement={onViewInStatement}
            state={state}
          />
        </div>
      </div>
      {!isLoading && statement && range && (
        <SqlExcerpt
          sheetName={sheetName}
          dimmed={state === "OUTDATED" || state === "UNAVAILABLE"}
          statement={statement}
          range={range}
        />
      )}
    </div>
  );
}

function AnchorStatePill({
  onViewInStatement,
  state,
}: {
  onViewInStatement?: () => void;
  state: AnchorState;
}) {
  const { t } = useTranslation();
  if (state === "CURRENT") {
    if (!onViewInStatement) return null;
    return (
      <Button
        className="h-auto shrink-0 p-0 text-xs"
        onClick={onViewInStatement}
        size="xs"
        appearance="link"
      >
        {t("plan.review.thread.anchor.view-in-statement")}
        <ExternalLink className="size-3" />
      </Button>
    );
  }
  if (state === "OUTDATED") {
    return (
      <Badge className="gap-x-1 px-2 text-xs" variant="warning">
        <History className="size-3" />
        {t("plan.review.thread.anchor.outdated")}
      </Badge>
    );
  }
  // The diff has not settled the comment yet; show nothing rather than a
  // state that may flip in a moment.
  if (state === "PENDING") return null;
  return (
    <Badge className="gap-x-1 px-2 text-xs" variant="default">
      <Ban className="size-3" />
      {t("plan.review.thread.anchor.statement-unavailable")}
    </Badge>
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
