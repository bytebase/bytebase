import { Popover as BasePopover } from "@base-ui/react/popover";
import type { ComponentProps } from "react";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { cn } from "@/lib/utils";

export const AgentPopover = BasePopover.Root;
export const AgentPopoverTrigger = BasePopover.Trigger;
export const AgentPopoverTitle = BasePopover.Title;
export const AgentPopoverClose = BasePopover.Close;

export function AgentPopoverContent({
  children,
  className,
  side = "top",
  align = "end",
  ...props
}: ComponentProps<typeof BasePopover.Popup> & {
  side?: ComponentProps<typeof BasePopover.Positioner>["side"];
  align?: ComponentProps<typeof BasePopover.Positioner>["align"];
}) {
  return (
    <BasePopover.Portal container={getLayerRoot("agent")}>
      <BasePopover.Positioner
        side={side}
        align={align}
        sideOffset={4}
        className={LAYER_SURFACE_CLASS}
      >
        <BasePopover.Popup
          className={cn(
            "max-h-[var(--available-height)] overflow-y-auto rounded-sm border border-control-border bg-background text-sm text-control shadow-md focus:outline-hidden",
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
