import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
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
      <Button
        appearance="secondary"
        size="md"
        aria-label={accessibleLabel}
        className={cn(
          "min-w-0 flex-1 justify-start rounded-t-sm text-left hover:bg-transparent focus-visible:ring-inset focus-visible:ring-offset-0",
          action && "pr-0"
        )}
        onClick={onSelect}
        type="button"
      >
        {children}
      </Button>
      {action}
    </div>
  );
}
