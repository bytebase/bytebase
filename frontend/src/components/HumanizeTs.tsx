import type { ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { useTimeReading } from "@/hooks/useTimeReading";
import {
  formatAbsoluteDateTime,
  formatCompactDateTime,
  formatOperationalDateTime,
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
  /** Unix timestamp in seconds. */
  ts: number;
  mode?: TimeDisplayMode;
  className?: string;
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

const fixed = (
  format: (tsMs: number) => string
): TimeReading<number, string> => ({
  read: format,
  nextChangeAt: () => Number.POSITIVE_INFINITY,
});

// Each mode's label reading, and the reading its tooltip restores: the full
// date-time for every reduced form, the age for the full one.
const MODES: Record<
  TimeDisplayMode,
  {
    label: TimeReading<number, string>;
    Hidden: ComponentType<{ tsMs: number }>;
  }
> = {
  queue: { label: queueTimeReading, Hidden: FullDateTime },
  compact: { label: fixed(formatCompactDateTime), Hidden: FullDateTime },
  operational: {
    label: fixed(formatOperationalDateTime),
    Hidden: FullDateTime,
  },
  datetime: { label: fixed(formatAbsoluteDateTime), Hidden: RelativeAge },
};

/**
 * Renders a timestamp in the form its surface calls for, and by default
 * reveals on hover the reading its label hides. This is the single canonical
 * way to display a record timestamp across the app.
 */
export function HumanizeTs({
  ts,
  mode = "queue",
  className,
  tooltip = true,
}: HumanizeTsProps) {
  // Subscribe to locale changes so the rendered strings update on a language switch.
  useTranslation();
  // An unvalidated string reaching `new Date(...)` gives NaN, which `Intl`
  // throws on; a row loses its timestamp rather than the page its subtree.
  const instantMs = ts * 1000;
  const tsMs = Number.isFinite(instantMs) ? instantMs : undefined;
  const { label: labelReading, Hidden } = MODES[mode];
  const label = useTimeReading(labelReading, tsMs);
  if (tsMs === undefined) {
    return null;
  }
  if (!tooltip) {
    return <span className={className}>{label}</span>;
  }
  return (
    <Tooltip
      content={<Hidden tsMs={tsMs} />}
      render={<span className={className} />}
    >
      {label}
    </Tooltip>
  );
}
