import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";

interface TooltipProps {
  readonly content: ReactNode;
  readonly children: ReactNode;
  readonly side?: "top" | "bottom" | "left" | "right";
  readonly delayDuration?: number;
}

export function Tooltip({
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
        <BaseTooltip.Portal>
          <BaseTooltip.Positioner side={side} sideOffset={4}>
            <BaseTooltip.Popup className="rounded-xs bg-gray-900 px-2.5 py-1.5 text-xs text-white shadow-md max-w-56">
              {content}
              <BaseTooltip.Arrow className="fill-gray-900" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}
