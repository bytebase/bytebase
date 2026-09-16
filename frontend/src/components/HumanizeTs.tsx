import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { useNow } from "@/hooks/useNow";
import {
  formatAbsoluteDateTime,
  formatCompactDateTime,
  formatOperationalDateTime,
  formatQueueTime,
  formatRelativeTime,
  nextRelativeChangeAt,
} from "@/utils";

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
   * Whether to reveal the absolute timestamp on hover. Defaults to true.
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

// Every reduced form hides the exact instant, so its tooltip restores it. The
// full form hides nothing but the age, so its tooltip supplies that instead.
const TOOLTIPS: Record<TimeDisplayMode, (tsMs: number) => string> = {
  queue: formatAbsoluteDateTime,
  compact: formatAbsoluteDateTime,
  operational: formatAbsoluteDateTime,
  datetime: formatRelativeTime,
};

/**
 * Renders a timestamp in the form its surface calls for, and by default
 * reveals the full date-time on hover. This is the single canonical way to
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
  // Only the work-queue form ages. The absolute modes render the same string
  // forever, so they take no place on the shared clock -- which matters on the
  // audit log, where thousands of full cells would otherwise subscribe for a
  // label that never moves. The formatters read the clock themselves; this
  // call is what brings the render back around.
  useNow(mode === "queue" ? nextRelativeChangeAt(tsMs) : undefined);
  const label = <span className={className}>{FORMATTERS[mode](tsMs)}</span>;
  if (!tooltip) {
    return label;
  }
  return <Tooltip content={TOOLTIPS[mode](tsMs)}>{label}</Tooltip>;
}
