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
 * and an indigo center dot. Done is a green fill, failed a red fill. The
 * remaining neutral states share one gray ring and are told apart by glyph
 * shape, so they stay distinguishable without relying on color (verify with a
 * grayscale / colorblind check).
 *
 * The ring border is a single 1.5px weight at every size, and each glyph's
 * stroke is tuned to render at that same weight — a lucide `strokeWidth` is in
 * the 24-unit viewBox, so it renders as `strokeWidth * glyph / 24` px, and
 * `1.5 * 24 / glyph` makes the glyph line match the ring.
 */
const BOX_CLASSES = {
  tiny: "size-4",
  small: "size-5",
  medium: "size-6",
  large: "size-7",
} as const;

// ~0.55 of the box so glyphs keep clear air around the ring at every size.
const GLYPH_CLASSES = {
  tiny: "size-2.5",
  small: "size-3",
  medium: "size-3.5",
  large: "size-4",
} as const;

// Glyph stroke rendered ≈ 1.5px to match the ring: 1.5 * 24 / glyph(10/12/14/16).
const STROKE_WIDTHS = {
  tiny: 3.6,
  small: 3,
  medium: 2.6,
  large: 2.25,
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
  const stroke = STROKE_WIDTHS[size];

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
          isNeutral && "border-[1.5px] border-control-light text-control-light",
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
          <Clock aria-hidden="true" className={glyph} strokeWidth={stroke} />
        )}
        {status === Task_Status.RUNNING && (
          <>
            <span
              aria-hidden="true"
              className="absolute inset-0 rounded-full border-[1.5px] border-control-light"
            />
            <span
              aria-hidden="true"
              className="absolute inset-0 rounded-full border-[1.5px] border-transparent border-t-accent motion-safe:animate-spin"
            />
            <span
              aria-hidden="true"
              className="h-2/5 w-2/5 rounded-full bg-accent"
            />
          </>
        )}
        {status === Task_Status.SKIPPED && (
          <SkipForward
            aria-hidden="true"
            className={glyph}
            strokeWidth={stroke}
          />
        )}
        {status === Task_Status.DONE && (
          <Check aria-hidden="true" className={glyph} strokeWidth={stroke} />
        )}
        {status === Task_Status.FAILED && (
          <X aria-hidden="true" className={glyph} strokeWidth={stroke} />
        )}
        {status === Task_Status.CANCELED && (
          <Ban aria-hidden="true" className={glyph} strokeWidth={stroke} />
        )}
      </div>
    </Tooltip>
  );
}
