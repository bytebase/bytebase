import { PreviewCard as BasePreviewCard } from "@base-ui/react/preview-card";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

// A hover-triggered card for read-only detail about the thing under the
// cursor. Unlike Tooltip it may contain structured content, and unlike Popover
// it opens on hover intent rather than click.
//
// Base UI opens the card on focus as well as hover, but it does not make the
// trigger focusable: it renders an <a> with no href, and callers overriding
// `render` usually supply a span. Triggers must set their own tabIndex, or the
// card is reachable by mouse only.

// ---- Root ----
const PreviewCard = BasePreviewCard.Root;

// ---- Trigger ----
const PreviewCardTrigger = BasePreviewCard.Trigger;

// ---- Portal + Positioner + Popup ----
function PreviewCardContent({
  className,
  children,
  side = "top",
  align = "start",
  sideOffset = 6,
  ref,
  ...props
}: ComponentProps<typeof BasePreviewCard.Popup> & {
  side?: ComponentProps<typeof BasePreviewCard.Positioner>["side"];
  align?: ComponentProps<typeof BasePreviewCard.Positioner>["align"];
  sideOffset?: ComponentProps<typeof BasePreviewCard.Positioner>["sideOffset"];
}) {
  return (
    <BasePreviewCard.Portal container={getLayerRoot("overlay")}>
      <BasePreviewCard.Positioner
        side={side}
        align={align}
        sideOffset={sideOffset}
        className={LAYER_SURFACE_CLASS}
      >
        <BasePreviewCard.Popup
          ref={ref}
          className={cn(
            "rounded-sm border border-control-border bg-background p-3 shadow-md text-sm text-control",
            "focus:outline-hidden",
            className
          )}
          {...props}
        >
          {children}
        </BasePreviewCard.Popup>
      </BasePreviewCard.Positioner>
    </BasePreviewCard.Portal>
  );
}

export { PreviewCard, PreviewCardContent, PreviewCardTrigger };
