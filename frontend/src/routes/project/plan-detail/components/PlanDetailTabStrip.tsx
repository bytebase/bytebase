import { Button as BaseButton } from "@base-ui/react/button";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function PlanDetailTabStrip({
  action,
  trailing,
  children,
}: {
  action?: ReactNode;
  trailing?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="relative bg-background pt-3">
      <div className="absolute bottom-0 w-full border-b border-b-block-border leading-0" />
      <div className="flex min-w-0 items-center gap-x-2">
        <div
          className={cn(
            "flex min-w-0 items-center overflow-x-auto",
            trailing ? "pl-4" : "flex-1 px-4"
          )}
        >
          {children}
        </div>
        {trailing && (
          <div className="flex shrink-0 items-center pr-4">{trailing}</div>
        )}
        {action && <div className="ml-auto shrink-0 px-4">{action}</div>}
      </div>
    </div>
  );
}

export function PlanDetailTabItem({
  accessibleLabel,
  action,
  boundedWidth = false,
  children,
  onSelect,
  selected,
}: {
  accessibleLabel?: string;
  action?: ReactNode;
  boundedWidth?: boolean;
  children: ReactNode;
  onSelect: () => void;
  selected: boolean;
}) {
  return (
    <div
      className={cn(
        // No transition here: the tab body swaps in the same commit, so a
        // color fade makes the highlight lag the content and read as flicker.
        "relative flex shrink-0 items-center rounded-t-sm border",
        action && "gap-1 pr-1",
        boundedWidth && "min-w-[min(10rem,100%)] max-w-[min(16rem,100%)]",
        selected
          ? "border-block-border border-b-transparent bg-background"
          : "border-b-block-border border-transparent hover:bg-control-bg"
      )}
    >
      <BaseButton
        aria-label={accessibleLabel}
        className={cn(
          "flex min-h-9 min-w-0 flex-1 cursor-pointer items-center gap-1.5 rounded-t-sm py-2 text-left text-sm font-medium leading-5 text-control focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent",
          action ? "pl-3" : "px-3"
        )}
        onClick={onSelect}
        type="button"
      >
        {children}
      </BaseButton>
      {action}
    </div>
  );
}
