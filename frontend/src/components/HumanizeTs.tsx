import type { ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { useTimeReading } from "@/hooks/useTimeReading";
import { cn } from "@/lib/utils";
import {
  absoluteTimeReading,
  compactTimeReading,
  type DateTimeSegments,
  dateTimeSegments,
  displayableInstantMs,
  formatAbsoluteDateTime,
  operationalTimeReading,
  queueTimeReading,
  relativeTimeReading,
  type TimeReading,
} from "@/utils/datetime";

/**
 * Which reading a surface supports, per `docs/design/timestamp-display.md`.
 *
 * - `queue` — freshness-first work queues and feeds: relative age, switching to
 *   an absolute date at 30 days.
 * - `compact` — history rows scanned to locate a record: date and time to the
 *   minute.
 * - `operational` — a future time the reader acts on, so the timezone is named
 *   in the visible string rather than left to a hover.
 * - `datetime` — a record whose time is the evidence: full precision inline.
 */
export type TimeDisplayMode = "queue" | "compact" | "operational" | "datetime";

interface HumanizeTsProps {
  /** The instant to show, in milliseconds; nothing renders without one. */
  tsMs: number | undefined;
  mode?: TimeDisplayMode;
  className?: string;
  /**
   * Keep the label on one line inside its box. Where the box is narrower than
   * the label, the time and zone ellipsize and the date stays whole.
   */
  truncate?: boolean;
  /**
   * Whether to reveal the hidden reading on hover. Defaults to true.
   * Disable only inside agent-layer overlays, where the shared Tooltip mounts
   * into the lower overlay layer (behind the agent window); supply an
   * AgentTooltip at the call site instead.
   */
  tooltip?: boolean;
}

// Tooltip bodies mount only while open, so as components they cost nothing
// for the rows nobody hovers, and a time-varying one can keep counting.
export function FullDateTime({ tsMs }: { tsMs: number }) {
  return <>{formatAbsoluteDateTime(tsMs)}</>;
}

function RelativeAge({ tsMs }: { tsMs: number }) {
  return <>{useTimeReading(relativeTimeReading, tsMs)}</>;
}

// Each mode's label reading, the reading its tooltip restores — the full
// date-time for every reduced form, the age for the full one — and whether
// its label carries a time that can give way to the date.
const MODES: Record<
  TimeDisplayMode,
  {
    label: TimeReading<number, string>;
    Hidden: ComponentType<{ tsMs: number }>;
    hasTime: boolean;
  }
> = {
  queue: { label: queueTimeReading, Hidden: FullDateTime, hasTime: false },
  compact: { label: compactTimeReading, Hidden: FullDateTime, hasTime: true },
  operational: {
    label: operationalTimeReading,
    Hidden: FullDateTime,
    hasTime: true,
  },
  datetime: { label: absoluteTimeReading, Hidden: RelativeAge, hasTime: true },
};

// The space where the time meets the date is kept out of the part that gives
// way, so a cut time never runs into the date, and is `whitespace-pre`
// because a flex item drops the space at its edge.
function DateKeptWhole({ date, rest, dateFirst }: DateTimeSegments) {
  const time = dateFirst ? rest.trimStart() : rest.trimEnd();
  const gap = dateFirst
    ? rest.slice(0, rest.length - time.length)
    : rest.slice(time.length);
  const pieces = [
    <span key="date" className="shrink-0">
      {date}
    </span>,
    gap ? (
      <span key="gap" className="shrink-0 whitespace-pre">
        {gap}
      </span>
    ) : null,
    <span key="time" className="min-w-0 truncate">
      {time}
    </span>,
  ];
  return <>{dateFirst ? pieces : pieces.reverse()}</>;
}

/**
 * Renders a timestamp in the form its surface calls for, and by default
 * reveals on hover the reading its label hides. This is the single canonical
 * way to display a record timestamp across the app.
 */
export function HumanizeTs({
  tsMs: instantMs,
  mode = "queue",
  className,
  truncate = false,
  tooltip = true,
}: HumanizeTsProps) {
  // Subscribe to locale changes so the rendered strings update on a language switch.
  useTranslation();
  // A row loses its timestamp rather than the page its subtree: what is not an
  // instant has no rendering, and the formatters throw on it.
  const tsMs = displayableInstantMs(instantMs);
  const { label: labelReading, Hidden, hasTime } = MODES[mode];
  const label = useTimeReading(labelReading, tsMs);
  if (tsMs === undefined) {
    return null;
  }
  const segments =
    truncate && hasTime && label !== undefined
      ? dateTimeSegments(label, tsMs)
      : undefined;
  const boxClassName = !truncate
    ? className
    : cn(
        segments ? "flex min-w-0 overflow-hidden" : "block truncate",
        className
      );
  const content = segments ? <DateKeptWhole {...segments} /> : label;
  if (!tooltip) {
    return <span className={boxClassName}>{content}</span>;
  }
  return (
    <Tooltip
      content={<Hidden tsMs={tsMs} />}
      render={<span className={boxClassName} />}
    >
      {content}
    </Tooltip>
  );
}
