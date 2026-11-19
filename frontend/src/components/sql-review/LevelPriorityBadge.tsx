import type { ReactNode } from "react";
import { Badge, type BadgeProps } from "@/components/ui/badge";

// SQL Review V1 levels are shown as the AI review's priorities: an error is
// P0, the SQL must change; a warning is P1, a person must accept it. Configs
// and results keep their levels; only the display changes.
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
