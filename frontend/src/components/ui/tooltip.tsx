import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

interface TooltipProps {
  readonly content: ReactNode;
  readonly children: ReactNode;
  readonly side?: "top" | "bottom" | "left" | "right";
  readonly delayDuration?: number;
  readonly popupClassName?: string;
  readonly open?: boolean;
  readonly onOpenChange?: ComponentProps<
    typeof BaseTooltip.Root
  >["onOpenChange"];
}

export function Tooltip({
  content,
  children,
  side = "top",
  delayDuration = 100,
  popupClassName,
  open,
  onOpenChange,
  render,
}: TooltipProps & {
  /** The trigger element; defaults to an inline-flex wrapper around `children`. */
  readonly render?: ComponentProps<typeof BaseTooltip.Trigger>["render"];
}) {
  if (!content) {
    return <>{children}</>;
  }

  return (
    <BaseTooltip.Root open={open} onOpenChange={onOpenChange}>
      <BaseTooltip.Trigger
        delay={delayDuration}
        render={render ?? <span className="inline-flex" />}
      >
        {children}
      </BaseTooltip.Trigger>
      <BaseTooltip.Portal container={getLayerRoot("overlay")}>
        <BaseTooltip.Positioner
          side={side}
          sideOffset={4}
          className={LAYER_SURFACE_CLASS}
        >
          <BaseTooltip.Popup
            className={cn(
              "max-w-56 rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md",
              popupClassName
            )}
          >
            {content}
            <BaseTooltip.Arrow className="fill-main" />
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}

/**
 * BlockTooltip is the same as Tooltip but renders the trigger as a block-level
 * div instead of an inline span. Use this when wrapping block content like
 * form sections.
 */
export function BlockTooltip({
  content,
  children,
  side = "top",
  delayDuration = 100,
  popupClassName,
  render,
  open,
  onOpenChange,
}: TooltipProps & {
  readonly render?: ComponentProps<typeof BaseTooltip.Trigger>["render"];
}) {
  if (!content) {
    return <>{children}</>;
  }

  return (
    <BaseTooltip.Root open={open} onOpenChange={onOpenChange}>
      <BaseTooltip.Trigger
        delay={delayDuration}
        render={render ?? <div className="flex-1 min-w-0" />}
      >
        {children}
      </BaseTooltip.Trigger>
      <BaseTooltip.Portal container={getLayerRoot("overlay")}>
        <BaseTooltip.Positioner
          side={side}
          sideOffset={4}
          className={LAYER_SURFACE_CLASS}
        >
          <BaseTooltip.Popup
            className={cn(
              "max-w-56 rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md",
              popupClassName
            )}
          >
            {content}
            <BaseTooltip.Arrow className="fill-main" />
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}
