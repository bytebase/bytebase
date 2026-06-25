import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { formatAbsoluteDateTime, formatRelativeTime } from "@/utils";

interface HumanizeTsProps {
  /** Unix timestamp in seconds. */
  ts: number;
  className?: string;
  /**
   * Whether to reveal the absolute timestamp on hover. Defaults to true.
   * Disable only inside agent-layer overlays, where the shared Tooltip mounts
   * into the lower overlay layer (behind the agent window); supply an
   * AgentTooltip at the call site instead.
   */
  tooltip?: boolean;
}

/**
 * Renders a relative timestamp ("5 minutes ago") and, by default, reveals the
 * absolute timestamp on hover. This is the single canonical way to display a
 * relative time across the app.
 */
export function HumanizeTs({ ts, className, tooltip = true }: HumanizeTsProps) {
  // Subscribe to locale changes so the rendered strings update on a language switch.
  useTranslation();
  const tsMs = ts * 1000;
  const label = <span className={className}>{formatRelativeTime(tsMs)}</span>;
  if (!tooltip) {
    return label;
  }
  return <Tooltip content={formatAbsoluteDateTime(tsMs)}>{label}</Tooltip>;
}
