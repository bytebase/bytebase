import { Check } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export type StepIndicatorStep = {
  title: ReactNode;
  key?: string | number;
};

export function StepIndicator({
  steps,
  currentIndex,
  currentKey,
  className,
}: {
  readonly steps: StepIndicatorStep[];
  readonly currentIndex?: number;
  readonly currentKey?: string | number;
  readonly className?: string;
}) {
  const activeIndex =
    currentIndex ??
    (currentKey === undefined
      ? 0
      : steps.findIndex((step) => step.key === currentKey));

  return (
    <ol
      data-slot="step-indicator"
      className={cn(
        "flex flex-wrap items-center gap-x-2 gap-y-1 px-0.5",
        className
      )}
    >
      {steps.map((step, index) => {
        const isReached = activeIndex >= 0 && index <= activeIndex;
        const isCompleted = activeIndex >= 0 && index < activeIndex;
        return (
          <li key={step.key ?? index} className="flex items-center gap-x-2">
            {index > 0 && <div className="w-8 h-px bg-control-border" />}
            <div className="flex items-center gap-x-2">
              <div
                className={cn(
                  "flex size-6 items-center justify-center rounded-full text-xs font-medium",
                  isReached
                    ? "bg-accent text-accent-text"
                    : "bg-control-bg-hover text-control-light"
                )}
              >
                {isCompleted ? <Check className="size-3.5" /> : index + 1}
              </div>
              <span
                className={cn(
                  "text-sm",
                  isReached ? "text-accent font-medium" : "text-control-light"
                )}
              >
                {step.title}
              </span>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
