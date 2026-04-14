import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";

interface TooltipProps {
  readonly content: ReactNode;
  readonly children: ReactNode;
  readonly side?: "top" | "bottom" | "left" | "right";
  readonly delayDuration?: number;
}

export function AgentTooltip({
  content,
  children,
  side = "top",
  delayDuration = 100,
}: TooltipProps) {
  if (!content) {
    return <>{children}</>;
  }

  return (
    <BaseTooltip.Provider delay={delayDuration}>
      <BaseTooltip.Root>
        <BaseTooltip.Trigger render={<span className="inline-flex" />}>
          {children}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal container={getLayerRoot("agent")}>
          <BaseTooltip.Positioner
            side={side}
            sideOffset={4}
            className={LAYER_SURFACE_CLASS}
          >
            <BaseTooltip.Popup className="max-w-56 rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md">
              {content}
              <BaseTooltip.Arrow className="fill-main" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}

export function AgentBlockTooltip({
  content,
  children,
  side = "top",
  delayDuration = 100,
}: TooltipProps) {
  if (!content) {
    return <>{children}</>;
  }

  return (
    <BaseTooltip.Provider delay={delayDuration}>
      <BaseTooltip.Root>
        <BaseTooltip.Trigger render={<div className="min-w-0 flex-1" />}>
          {children}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal container={getLayerRoot("agent")}>
          <BaseTooltip.Positioner
            side={side}
            sideOffset={4}
            className={LAYER_SURFACE_CLASS}
          >
            <BaseTooltip.Popup className="max-w-56 rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md">
              {content}
              <BaseTooltip.Arrow className="fill-main" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}
