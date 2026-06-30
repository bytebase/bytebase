import type { ReactNode } from "react";
import { cn } from "@/react/lib/utils";

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
    <div className="relative bg-white pt-3">
      <div className="absolute bottom-0 w-full border-b border-b-gray-200 leading-0" />
      <div className="flex items-center justify-between gap-x-4">
        <div
          className={cn(
            "flex min-w-0 items-center overflow-x-auto",
            // With a trailing slot the container shrinks to its tabs so the
            // sticky button sits right after the last tab (and pins to the
            // edge on overflow); the occluder below supplies the right
            // padding. Otherwise fill the row with symmetric padding.
            trailing ? "pl-4" : "flex-1 px-4"
          )}
        >
          {children}
          {trailing && (
            <div className="sticky right-0 ml-1 flex shrink-0 items-center self-stretch border-b border-b-gray-200 bg-white pr-4 pl-2">
              {trailing}
            </div>
          )}
        </div>
        {action && <div className="shrink-0 px-4">{action}</div>}
      </div>
    </div>
  );
}

export function PlanDetailTabItem({
  action,
  children,
  onSelect,
  selected,
}: {
  action?: ReactNode;
  children: ReactNode;
  onSelect: () => void;
  selected: boolean;
}) {
  return (
    <div
      className={cn(
        "relative flex shrink-0 items-center rounded-t-lg border transition-all",
        selected
          ? "border-gray-200 border-b-transparent bg-white"
          : "border-b-gray-200 border-transparent hover:opacity-80"
      )}
    >
      <button
        className="flex min-h-9 shrink-0 items-center gap-2 px-4 py-2 text-left"
        onClick={onSelect}
        type="button"
      >
        {children}
      </button>
      {action}
    </div>
  );
}
