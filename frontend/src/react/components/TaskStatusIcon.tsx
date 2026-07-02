import { Ban, Check, Clock, SkipForward, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";

/**
 * Canonical status icon for a rollout task/stage. Drive the size with the
 * `size` prop. Use this everywhere a `Task_Status` is shown so the plan table,
 * plan detail task rows, and stage tabs stay visually consistent.
 *
 * Color only carries meaning for the states that need attention. Running is a
 * spinner built as the neutral ring "coming alive": the same gray ring track
 * the pending/skipped/canceled states use, with an indigo arc sweeping over it
 * (the track-plus-arc construction Material and GitLab use, and the mainstream
 * CI/CD convention for in-progress). Done is a green fill, failed a red fill.
 * The remaining neutral states share one gray ring and are told apart by glyph
 * shape, so they stay distinguishable without relying on color (verify with a
 * grayscale / colorblind check).
 *
 * Box, ring thickness, and glyph size all scale with `size` so the badge stays
 * balanced from the 16px stage tab up to the 28px detail view.
 */
const BOX_CLASSES = {
  tiny: "size-4",
  small: "size-5",
  medium: "size-6",
  large: "size-7",
} as const;

const BORDER_CLASSES = {
  tiny: "border-[1.5px]",
  small: "border-2",
  medium: "border-2",
  large: "border-2",
} as const;

// ~0.55 of the box so glyphs keep clear air around the ring at every size.
const GLYPH_CLASSES = {
  tiny: "size-2.5",
  small: "size-3",
  medium: "size-3.5",
  large: "size-4",
} as const;

export function TaskStatusIcon({
  status,
  size = "small",
}: {
  status: Task_Status;
  size?: keyof typeof BOX_CLASSES;
}) {
  const { t } = useTranslation();
  const statusLabel = stringifyTaskStatus(status, t);
  const glyph = GLYPH_CLASSES[size];
  const border = BORDER_CLASSES[size];

  // Neutral states share one gray ring and are distinguished by their glyph,
  // never by color alone.
  const isNeutral =
    status === Task_Status.STATUS_UNSPECIFIED ||
    status === Task_Status.NOT_STARTED ||
    status === Task_Status.PENDING ||
    status === Task_Status.SKIPPED ||
    status === Task_Status.CANCELED;

  return (
    <Tooltip content={statusLabel}>
      <div
        role="img"
        aria-label={statusLabel}
        className={cn(
          "relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full select-none",
          BOX_CLASSES[size],
          (isNeutral || status === Task_Status.RUNNING) && "bg-white",
          isNeutral && cn(border, "border-control-light text-control-light"),
          status === Task_Status.STATUS_UNSPECIFIED && "border-dashed",
          status === Task_Status.DONE && "bg-success text-white",
          status === Task_Status.FAILED && "bg-error text-white"
        )}
      >
        {status === Task_Status.NOT_STARTED && (
          <span
            aria-hidden="true"
            className="h-2/5 w-2/5 rounded-full bg-current"
          />
        )}
        {status === Task_Status.PENDING && (
          <Clock aria-hidden="true" className={glyph} />
        )}
        {status === Task_Status.RUNNING && (
          <>
            <span
              aria-hidden="true"
              className={cn(
                "absolute inset-0 rounded-full border-control-light",
                border
              )}
            />
            <span
              aria-hidden="true"
              className={cn(
                "absolute inset-0 rounded-full border-transparent border-t-accent motion-safe:animate-spin",
                border
              )}
            />
            <span
              aria-hidden="true"
              className="h-2/5 w-2/5 rounded-full bg-accent"
            />
          </>
        )}
        {status === Task_Status.SKIPPED && (
          <SkipForward aria-hidden="true" className={glyph} />
        )}
        {status === Task_Status.DONE && (
          <Check aria-hidden="true" className={glyph} strokeWidth={2.5} />
        )}
        {status === Task_Status.FAILED && (
          <X aria-hidden="true" className={glyph} strokeWidth={2.5} />
        )}
        {status === Task_Status.CANCELED && (
          <Ban aria-hidden="true" className={glyph} />
        )}
      </div>
    </Tooltip>
  );
}
