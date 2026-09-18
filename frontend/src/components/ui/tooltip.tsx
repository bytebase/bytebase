import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ComponentProps, ReactElement, ReactNode } from "react";
import { cloneElement } from "react";
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
  /**
   * The trigger element; defaults to an inline-flex wrapper around `children`.
   * An element rather than Base UI's wider render type, because this one is
   * kept even when there is no tooltip, and a render *function* has nothing to
   * be called with in that state.
   */
  readonly render?: ReactElement;
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
}: TooltipProps) {
  if (!content) {
    // A caller's `render` element carries that row's layout, not the tooltip's:
    // dropping it when there is nothing to say would move their classes onto a
    // grandchild in one state and not the other, which is a layout that breaks
    // only sometimes.
    return render ? cloneElement(render, undefined, children) : <>{children}</>;
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
export function BlockTooltip({ render, ...props }: TooltipProps) {
  // A caller's own element is kept in both states; this default is the
  // tooltip's, so it comes and goes with the tooltip.
  const blockTrigger = props.content ? (
    <div className="flex-1 min-w-0" />
  ) : undefined;
  return <Tooltip {...props} render={render ?? blockTrigger} />;
}
