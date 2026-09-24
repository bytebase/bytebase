import { ChevronDown, ChevronRight } from "lucide-react";
import {
  type CSSProperties,
  type MouseEvent,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FullDateTime } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { displayableInstantMs } from "@/utils/datetime";
import type { DisplayItem, Section } from "./types";

// The full reading is offered only for an instant the formatters can render.
const fullTime = (timeMs: number | undefined) => {
  const instantMs = displayableInstantMs(timeMs);
  return instantMs === undefined ? undefined : (
    <FullDateTime tsMs={instantMs} />
  );
};

// A row is an `xs` control (24px) inside `py-0.5`.
const ITEM_HEIGHT = 28;
const MAX_VISIBLE_ITEMS = 10;
const MAX_RENDERED_ITEMS = 50;
// Fixed: an unfolded block scrolls inside the box rather than growing it, so
// the sections below never move when a row is toggled.
const MAX_HEIGHT = `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px`;
const INDEX_COLUMN_MIN_WIDTH = "24px";
// Fits every relative time below 100 seconds ("+99.99s").
const RELATIVE_TIME_COLUMN_MIN_CHARS = 7;

export interface SectionContentProps {
  section: Section;
  indent?: boolean;
  datasetKey?: string;
  // The reader's explicit folds, by row key. Held by the viewer because this
  // component unmounts whenever its enclosing section collapses.
  foldOverrides: ReadonlyMap<string, boolean>;
  onFoldChange: (key: string, open: boolean) => void;
}

interface Row {
  item: DisplayItem;
  // Position in the section, which the sparse window below does not preserve.
  index: number;
}

// Nothing opens by default: the reader's folds are the only open state.
const isOpen = (
  item: DisplayItem,
  foldOverrides: ReadonlyMap<string, boolean>
): boolean => foldOverrides.get(item.key) ?? false;

// A statement recovered from a sheet range is a split segment, which usually
// begins with the line break that followed the previous statement.
const trimBlankEdges = (statement: string): string =>
  statement.replace(/^(?:[ \t]*\r?\n)+/, "").trimEnd();

// A ref callback's body: track the element until React detaches it.
const register = (
  elements: Map<string, HTMLElement>,
  key: string,
  element: HTMLElement
) => {
  elements.set(key, element);
  return () => {
    elements.delete(key);
  };
};

// An unfolded row near the bottom of the box opens its block below the visible
// edge, where a click would seem to have done nothing. Scroll the box by the
// least that shows the row: to its bottom edge when it fits, to its top edge —
// the line at the top, the block filling the rest — when it is taller than the
// box. The box's own scroll only: the block is clipped by the box, so a page
// scroll could not reveal it and would only move the log under the reader.
// Folding needs nothing: the browser's clamp puts the line back where the
// reveal took it from.
const revealRow = (box: HTMLElement, row: HTMLElement) => {
  const boxRect = box.getBoundingClientRect();
  const rowRect = row.getBoundingClientRect();
  const overflow = rowRect.bottom - boxRect.bottom;
  const headroom = rowRect.top - boxRect.top;
  box.scrollTop += Math.max(0, Math.min(overflow, headroom));
};

const sameKeys = (a: ReadonlySet<string>, b: ReadonlySet<string>): boolean => {
  if (a.size !== b.size) return false;
  for (const key of a) {
    if (!b.has(key)) return false;
  }
  return true;
};

export function SectionContent({
  section,
  indent = false,
  datasetKey,
  foldOverrides,
  onFoldChange,
}: SectionContentProps) {
  const { t } = useTranslation();
  const [showAllItems, setShowAllItems] = useState(false);
  const [clampedKeys, setClampedKeys] = useState<ReadonlySet<string>>(
    () => new Set()
  );
  const boxRef = useRef<HTMLDivElement>(null);
  const rowElements = useRef(new Map<string, HTMLElement>());
  const lineElements = useRef(new Map<string, HTMLElement>());
  const registerRow = useCallback(
    (key: string, element: HTMLElement) =>
      register(rowElements.current, key, element),
    []
  );
  const registerLine = useCallback(
    (key: string, element: HTMLElement) =>
      register(lineElements.current, key, element),
    []
  );

  useEffect(() => {
    setShowAllItems(false);
  }, [datasetKey, section.id]);

  // The first MAX_RENDERED_ITEMS rows, plus the marked row when it falls
  // outside them: a failure at statement 300 must not wait behind "Load more".
  const { head, tail } = useMemo(() => {
    const rows: Row[] = section.items.map((item, index) => ({ item, index }));
    if (showAllItems || rows.length <= MAX_RENDERED_ITEMS) {
      return { head: rows, tail: [] };
    }
    const marked = rows.find(
      (row) => row.item.marked && row.index >= MAX_RENDERED_ITEMS
    );
    return {
      head: rows.slice(0, MAX_RENDERED_ITEMS),
      tail: marked ? [marked] : [],
    };
  }, [section.items, showAllItems]);
  const hiddenItemCount = section.items.length - head.length - tail.length;

  const markedKey = useMemo(
    () => section.items.find((item) => item.marked)?.key,
    [section.items]
  );

  // Which lines their width clamps. A row with no line to measure is open, and
  // keeps its verdict: it lost the line by opening, and dropping the verdict
  // would take away the control that closes it.
  const measureClampedLines = useCallback(() => {
    setClampedKeys((previous) => {
      const next = new Set<string>();
      for (const key of previous) {
        if (!lineElements.current.has(key) && rowElements.current.has(key)) {
          next.add(key);
        }
      }
      for (const [key, line] of lineElements.current) {
        if (line.scrollWidth > line.clientWidth) next.add(key);
      }
      return sameKeys(previous, next) ? previous : next;
    });
  }, []);

  useLayoutEffect(() => {
    const box = boxRef.current;
    if (!box) return;
    const observer = new ResizeObserver(measureClampedLines);
    observer.observe(box);
    return () => observer.disconnect();
  }, [measureClampedLines]);

  // The box is capped, so rows arriving by "Load more" or a poll grow its
  // scroll height and never resize it: the observer alone would miss them.
  // The dependencies are everything that changes which lines are rendered or
  // how wide they are.
  useLayoutEffect(() => {
    measureClampedLines();
  }, [measureClampedLines, head, tail, foldOverrides, indent]);

  // The row the reader just unfolded, revealed by the layout effect below
  // once its block is laid out and before it is painted.
  const unfoldedKey = useRef<string | undefined>(undefined);
  const handleToggle = useCallback(
    (key: string, open: boolean) => {
      unfoldedKey.current = open ? key : undefined;
      onFoldChange(key, open);
    },
    [onFoldChange]
  );
  // foldOverrides changes for a toggle in any section; only the section that
  // recorded the unfold acts on it.
  useLayoutEffect(() => {
    const key = unfoldedKey.current;
    unfoldedKey.current = undefined;
    const box = boxRef.current;
    const row = key ? rowElements.current.get(key) : undefined;
    if (box && row) revealRow(box, row);
  }, [foldOverrides]);

  // Bring the marked row into view once, when the mark lands on it or on mount.
  // The box's own scrollTop: scrollIntoView would also scroll the page under a
  // reader who is looking at something else.
  useLayoutEffect(() => {
    const box = boxRef.current;
    const row = markedKey ? rowElements.current.get(markedKey) : undefined;
    if (!box || !row) return;
    const top = row.offsetTop;
    const isInView =
      top >= box.scrollTop &&
      top + row.offsetHeight <= box.scrollTop + box.clientHeight;
    if (!isInView) box.scrollTop = top;
  }, [markedKey]);

  const rendered = [...head, ...tail];
  // Both numeric columns are sized once, from the widest value this section
  // draws. A per-row minimum would let one wide row push its own statement
  // right of its neighbours'.
  const indexChars = String((rendered.at(-1)?.index ?? 0) + 1).length;
  const relativeTimeChars = rendered.reduce(
    (widest, { item }) => Math.max(widest, item.relativeTime.length),
    RELATIVE_TIME_COLUMN_MIN_CHARS
  );

  const renderRow = ({ item, index }: Row) => {
    const open = isOpen(item, foldOverrides);
    return (
      <LogRow
        key={item.key}
        item={item}
        index={index}
        indent={indent}
        open={open}
        isClamped={clampedKeys.has(item.key)}
        onToggle={() => handleToggle(item.key, !open)}
        registerRow={registerRow}
        registerLine={registerLine}
      />
    );
  };

  return (
    <div
      ref={boxRef}
      className="relative overflow-auto border-block-border border-t bg-control-bg/50"
      style={
        {
          maxHeight: MAX_HEIGHT,
          "--task-log-index-width": `max(${INDEX_COLUMN_MIN_WIDTH}, ${indexChars}ch)`,
          "--task-log-time-width": `${relativeTimeChars}ch`,
        } as CSSProperties
      }
    >
      {head.map(renderRow)}
      {hiddenItemCount > 0 ? (
        <Button
          type="button"
          appearance="secondary"
          size="sm"
          className="w-full rounded-none border-block-border border-t text-control-light hover:bg-control-bg-hover hover:text-control"
          onClick={() => setShowAllItems(true)}
        >
          <span>{t("common.load-more")}</span>
          <span className="tabular-nums">({hiddenItemCount})</span>
        </Button>
      ) : null}
      {tail.map(renderRow)}
    </div>
  );
}

interface LogRowProps {
  item: DisplayItem;
  index: number;
  indent: boolean;
  open: boolean;
  isClamped: boolean;
  onToggle: () => void;
  registerRow: (key: string, element: HTMLElement) => () => void;
  registerLine: (key: string, element: HTMLElement) => () => void;
}

const TEXT_CELL = "shrink-0 py-1 tabular-nums";

function LogRow({
  item,
  index,
  indent,
  open,
  isClamped,
  onToggle,
  registerRow,
  registerLine,
}: LogRowProps) {
  const { t } = useTranslation();
  const { statement, error } = item;
  const showBlock = open && statement !== undefined;
  const hasBothPayloads = statement !== undefined && error !== undefined;
  const lineIsStatement = statement !== undefined && error === undefined;
  // Unfolding must show something the line does not. With both payloads the
  // line is the error, so the statement is never on it. Otherwise the line is
  // the statement, and hides either its formatting or what its width cut off.
  // An open row always qualifies: what was opened must be closable, and its
  // clamp verdict does not survive this component remounting.
  const isFoldable =
    hasBothPayloads ||
    (lineIsStatement &&
      (open || statement.trim() !== item.detail || isClamped));
  const copyStatement = () => statement ?? "";

  // The whole line toggles, the statement and the error included, so a reader
  // never has to learn which part of a row is live. The unfolded block does
  // not: it is the content being read and selected from, and a double-click's
  // first click — indistinguishable from a single one — would remove the very
  // text being selected.
  const handleRowClick = (event: MouseEvent<HTMLDivElement>) => {
    const target = event.target as Element;
    if (target.closest("button, [data-log-payload='block']")) return;
    // A click that ends a drag, e.g. a selection swept across the clamped line.
    if (!(window.getSelection()?.isCollapsed ?? true)) return;
    onToggle();
  };

  return (
    <div
      ref={(element) => (element ? registerRow(item.key, element) : undefined)}
      data-testid="task-run-log-row"
      className={cn(
        "group flex items-start gap-x-2 py-0.5 hover:bg-control-bg",
        indent ? "px-6" : "px-3",
        index > 0 && "border-block-border border-t",
        isFoldable && "cursor-pointer"
      )}
      onClick={isFoldable ? handleRowClick : undefined}
    >
      <span
        className={cn(
          TEXT_CELL,
          "w-(--task-log-index-width) text-right text-control-placeholder"
        )}
      >
        {index + 1}
      </span>
      <span className={cn(TEXT_CELL, "text-control-placeholder")}>
        <Tooltip content={fullTime(item.timeMs)}>{item.time}</Tooltip>
      </span>
      <span
        className={cn(
          TEXT_CELL,
          "w-(--task-log-time-width) text-right text-control-placeholder"
        )}
      >
        {item.relativeTime}
      </span>
      <span className={cn(TEXT_CELL, item.levelClass)}>
        {item.levelIndicator}
      </span>
      {/* Reserved on every row so the statement column stays straight. */}
      <span className="flex w-6 shrink-0 justify-center">
        {isFoldable ? (
          <Button
            type="button"
            appearance="secondary"
            size="xs"
            className="px-1 text-control-light hover:bg-control-bg-hover hover:text-control"
            aria-expanded={showBlock}
            aria-label={
              showBlock
                ? t("task-run.log-detail.hide-full-statement")
                : t("task-run.log-detail.show-full-statement")
            }
            onClick={onToggle}
          >
            {showBlock ? (
              <ChevronDown className="size-3.5" />
            ) : (
              <ChevronRight className="size-3.5" />
            )}
          </Button>
        ) : null}
      </span>
      <div className="flex min-w-0 flex-1 flex-col items-start">
        {error !== undefined ? (
          <span
            data-log-payload="error"
            className={cn("max-w-full py-1 break-words", item.detailClass)}
          >
            {item.detail}
          </span>
        ) : statement === undefined ? (
          <span className={cn("max-w-full py-1 break-words", item.detailClass)}>
            {item.detail}
          </span>
        ) : showBlock ? null : (
          <span
            ref={(element) =>
              element ? registerLine(item.key, element) : undefined
            }
            data-log-payload="line"
            className={cn("max-w-full truncate py-1", item.detailClass)}
          >
            {item.detail}
          </span>
        )}
        {showBlock ? (
          <div
            data-log-payload="block"
            className={cn(
              "flex w-full cursor-text items-start gap-x-2 rounded-sm border border-block-border bg-background py-1 pr-1 pl-2.5 text-control",
              error !== undefined ? "mb-1" : "my-1"
            )}
          >
            <span
              data-log-sql
              className="min-w-0 flex-1 py-1 whitespace-pre-wrap break-words"
            >
              {trimBlankEdges(statement)}
            </span>
            {/* Sticky to the section's scroll box, so a statement taller than
                the box keeps its copy control in reach while it scrolls. */}
            <span data-log-copy className="sticky top-1 shrink-0">
              <CopyButton
                content={copyStatement}
                className="bg-control-bg text-control-light hover:bg-control-bg-hover hover:text-control"
              />
            </span>
          </div>
        ) : null}
      </div>
      <span className="ml-auto flex shrink-0 items-center gap-x-2">
        {lineIsStatement && !showBlock ? (
          <CopyButton
            content={copyStatement}
            className="bg-control-bg-hover text-control-light opacity-0 hover:text-control focus-visible:opacity-100 group-hover:opacity-100 pointer-coarse:opacity-100"
          />
        ) : null}
        {item.duration ? (
          <span className="py-1 text-info tabular-nums">{item.duration}</span>
        ) : null}
        {item.affectedRows !== undefined ? (
          <span className="py-1 text-control-placeholder">
            {item.affectedRows} {t("task.affected-rows")}
          </span>
        ) : null}
      </span>
    </div>
  );
}

export default SectionContent;
