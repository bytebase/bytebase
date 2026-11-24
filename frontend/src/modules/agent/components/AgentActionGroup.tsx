import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export function AgentActionGroup({
  className,
  ...props
}: ComponentProps<"div">) {
  return (
    <div
      {...props}
      className={cn(
        "flex w-full min-w-0 max-w-full flex-wrap items-center justify-end gap-2",
        className
      )}
      data-agent-action-group
    />
  );
}
