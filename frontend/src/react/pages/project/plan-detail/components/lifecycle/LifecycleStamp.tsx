import type { ReactNode } from "react";
import { cn } from "@/react/lib/utils";

// A bordered pill, not a button — it makes the lifecycle state visible without
// implying another primary action. Two sizes: `sm` (default) is a compact badge
// for the terminal state next to the title (Closed / Deployed); `md` matches the
// action button height (h-9) for right-slot statuses that stand in for the
// advance action, so the slot stays consistent when it swaps action ↔ status.
// The outline-not-fill, rounded-full shape keeps it reading as status either way.
export function LifecycleStamp({
  tone = "neutral",
  size = "sm",
  className,
  children,
}: {
  tone?: "neutral" | "success" | "error";
  size?: "sm" | "md";
  className?: string;
  children: ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded-full border text-sm",
        size === "sm" && "gap-x-1 px-2 py-0.5",
        size === "md" && "h-9 gap-x-1.5 px-3",
        tone === "success" && "border-success/40 text-success",
        tone === "error" && "border-error/40 text-error",
        tone === "neutral" && "text-control",
        className
      )}
    >
      {children}
    </span>
  );
}
