import type { ReactNode } from "react";
import { cn } from "@/react/lib/utils";

export function PlanDetailTabStrip({
  action,
  children,
}: {
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="relative bg-white pt-3">
      <div className="absolute bottom-0 w-full border-b border-b-gray-200 leading-0" />
      <div className="flex items-center justify-between gap-x-4">
        <div className="flex min-w-0 flex-1 items-center overflow-x-auto px-4">
          {children}
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
