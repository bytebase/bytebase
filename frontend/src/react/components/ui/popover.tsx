import { Popover as BasePopover } from "@base-ui/react/popover";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

// ---- Root ----
const Popover = BasePopover.Root;

// ---- Trigger ----
const PopoverTrigger = BasePopover.Trigger;

// ---- Portal + Positioner + Popup ----
function PopoverContent({
  className,
  children,
  side = "bottom",
  align = "end",
  sideOffset = 4,
  anchor,
  ref,
  ...props
}: ComponentProps<typeof BasePopover.Popup> & {
  side?: ComponentProps<typeof BasePopover.Positioner>["side"];
  align?: ComponentProps<typeof BasePopover.Positioner>["align"];
  sideOffset?: ComponentProps<typeof BasePopover.Positioner>["sideOffset"];
  anchor?: ComponentProps<typeof BasePopover.Positioner>["anchor"];
}) {
  return (
    <BasePopover.Portal container={getLayerRoot("overlay")}>
      <BasePopover.Positioner
        side={side}
        align={align}
        sideOffset={sideOffset}
        anchor={anchor}
        className={LAYER_SURFACE_CLASS}
      >
        <BasePopover.Popup
          ref={ref}
          className={cn(
            "rounded-sm border border-control-border bg-background p-3 shadow-md text-sm text-control",
            "focus:outline-hidden",
            className
          )}
          {...props}
        >
          {children}
        </BasePopover.Popup>
      </BasePopover.Positioner>
    </BasePopover.Portal>
  );
}

export { Popover, PopoverContent, PopoverTrigger };
