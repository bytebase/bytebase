import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { useNow } from "@/hooks/useNow";
import {
  formatAbsoluteDateTime,
  formatCompactDateTime,
  formatOperationalDateTime,
  formatQueueTime,
  formatRelativeTime,
  nextQueueTimeChangeAt,
  nextRelativeTimeChangeAt,
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
  /** Applied to the element that renders the time, which is the only one. */
  className?: string;
  /**
   * Whether to reveal the hidden reading on hover. Defaults to true.
   * Disable only inside agent-layer overlays, where the shared Tooltip mounts
   * into the lower overlay layer (behind the agent window); supply an
   * AgentTooltip at the call site instead.
   */
  tooltip?: boolean;
}

const FORMATTERS: Record<TimeDisplayMode, (tsMs: number) => string> = {
  queue: formatQueueTime,
  compact: formatCompactDateTime,
  operational: formatOperationalDateTime,
  datetime: formatAbsoluteDateTime,
};

// Tooltip bodies mount only while open, so as components they cost nothing
// for the rows nobody hovers, and a time-varying one can keep counting.
function FullDateTime({ tsMs }: { tsMs: number }) {
  return <>{formatAbsoluteDateTime(tsMs)}</>;
}

function RelativeAge({ tsMs }: { tsMs: number }) {
  useNow(nextRelativeTimeChangeAt(tsMs));
  return <>{formatRelativeTime(tsMs)}</>;
}

/**
 * Renders a timestamp in the form its surface calls for, and by default
 * reveals on hover the reading its label hides: the full date-time for every
 * reduced form, the age for the full one. This is the single canonical way to
 * display a record timestamp across the app.
 */
export function HumanizeTs({
  ts,
  mode = "queue",
  className,
  tooltip = true,
}: HumanizeTsProps) {
  // Subscribe to locale changes so the rendered strings update on a language switch.
  useTranslation();
  const tsMs = ts * 1000;
  // Only the work-queue label ages; the absolute labels never change, so they
  // take no place on the shared clock.
  useNow(mode === "queue" ? nextQueueTimeChangeAt(tsMs) : undefined);
  const label = FORMATTERS[mode](tsMs);
  if (!tooltip) {
    return <span className={className}>{label}</span>;
  }
  return (
    <Tooltip
      content={
        mode === "datetime" ? (
          <RelativeAge tsMs={tsMs} />
        ) : (
          <FullDateTime tsMs={tsMs} />
        )
      }
      render={<span className={className} />}
    >
      {label}
    </Tooltip>
  );
}
