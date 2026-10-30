import { Progress as BaseProgress } from "@base-ui/react/progress";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface ProgressProps {
  /** Progress toward `max`; null shows an indeterminate bar. */
  value: number | null;
  max?: number;
  /** Visible text above the bar that also names it for assistive technology. */
  label: ReactNode;
  className?: string;
}

function Progress({ value, max = 100, label, className }: ProgressProps) {
  return (
    <BaseProgress.Root
      value={value}
      max={max}
      className={cn("flex w-full flex-col gap-y-1.5", className)}
    >
      <BaseProgress.Label className="text-xs text-control-light">
        {label}
      </BaseProgress.Label>
      <BaseProgress.Track className="relative h-1 w-full overflow-hidden rounded-xs bg-control-bg">
        <BaseProgress.Indicator className="h-full bg-accent transition-all duration-300 data-indeterminate:w-1/3 data-indeterminate:animate-progress-indeterminate" />
      </BaseProgress.Track>
    </BaseProgress.Root>
  );
}

export { Progress };
