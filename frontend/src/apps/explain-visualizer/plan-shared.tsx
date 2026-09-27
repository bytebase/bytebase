import { cn } from "@/lib/utils";

/**
 * Small pieces the diagram, grid, summary and detail pane all draw, kept here
 * so the four surfaces cannot drift into showing the same thing differently.
 */

/** Estimates a phone-width reader can give up to keep the identity readable. */
export const SECONDARY_COLUMN_CLASS = "hidden sm:table-cell";

/**
 * A share of the plan's estimated cost, drawn as a proportion of a track.
 *
 * Decoration: every caller prints the same share, or the cost behind it, as
 * text beside the bar. `className` sizes the track; its height is fixed so the
 * surfaces stay comparable.
 */
export function PlanCostShareBar({
  share,
  className,
}: {
  share: number;
  className?: string;
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "h-1.5 overflow-hidden rounded-full bg-control-bg",
        className
      )}
    >
      <span
        data-testid="plan-cost-share-bar"
        className="block h-full rounded-full bg-warning"
        style={{ width: `${share * 100}%` }}
      />
    </span>
  );
}

/** One labelled estimate in a `<dl>` of them. */
export function PlanMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-w-32 flex-1 flex-col gap-1">
      <dt className="text-xs leading-4 text-control-light">{label}</dt>
      <dd className="text-sm leading-5 text-main tabular-nums">{value}</dd>
    </div>
  );
}
