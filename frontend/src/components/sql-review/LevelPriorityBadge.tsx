import type { ReactNode } from "react";
import { Badge, type BadgeProps } from "@/components/ui/badge";

// SQL Review V1 levels are shown as the AI review's priorities, an error as P0
// and a warning as P1. Only the display changes: an error still holds the
// rollout and a warning does not, and configs and results keep their levels.
export type ReviewLevel = "error" | "warning";

const PRIORITIES: Record<
  ReviewLevel,
  { label: string; variant: BadgeProps["variant"] }
> = {
  error: { label: "P0", variant: "destructive" },
  warning: { label: "P1", variant: "warning" },
};

export const levelPriority = (level: ReviewLevel) => PRIORITIES[level].label;

export function LevelPriorityBadge({
  level,
  className,
  children,
}: {
  level: ReviewLevel;
  className?: string;
  children?: ReactNode;
}) {
  const { label, variant } = PRIORITIES[level];
  return (
    <Badge className={className} variant={variant}>
      {label}
      {children}
    </Badge>
  );
}
