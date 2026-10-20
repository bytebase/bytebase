import { Check } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
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
  fitOneRow = false,
}: {
  readonly steps: StepIndicatorStep[];
  readonly currentIndex?: number;
  readonly currentKey?: string | number;
  readonly className?: string;
  // When the titles do not fit one row, only the current step keeps its title
  // in sight; the others stay for screen readers.
  readonly fitOneRow?: boolean;
}) {
  const activeIndex =
    currentIndex ??
    (currentKey === undefined
      ? 0
      : steps.findIndex((step) => step.key === currentKey));

  const listRef = useRef<HTMLOListElement>(null);
  // The steps laid out in full on one row, out of sight, to measure.
  const rowRef = useRef<HTMLOListElement>(null);
  const [narrow, setNarrow] = useState(false);
  useLayoutEffect(() => {
    const list = listRef.current;
    const row = rowRef.current;
    if (!fitOneRow || !list || !row || typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(() =>
      setNarrow(row.scrollWidth > list.clientWidth)
    );
    observer.observe(list);
    observer.observe(row);
    return () => observer.disconnect();
  }, [fitOneRow]);

  const items = (compact: boolean) =>
    steps.map((step, index) => {
      const isReached = activeIndex >= 0 && index <= activeIndex;
      const isCompleted = activeIndex >= 0 && index < activeIndex;
      return (
        <li key={step.key ?? index} className="flex items-center gap-x-2">
          {index > 0 && (
            <div
              className={cn("h-px bg-control-border", compact ? "w-4" : "w-8")}
            />
          )}
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
                isReached ? "text-accent font-medium" : "text-control-light",
                compact && index !== activeIndex && "sr-only"
              )}
            >
              {step.title}
            </span>
          </div>
        </li>
      );
    });

  const list = (
    <ol
      ref={listRef}
      data-slot="step-indicator"
      className={cn(
        "flex flex-wrap items-center gap-x-2 gap-y-1 px-0.5",
        className
      )}
    >
      {items(narrow)}
    </ol>
  );
  if (!fitOneRow) return list;

  return (
    <div className="relative">
      {list}
      <ol
        ref={rowRef}
        aria-hidden="true"
        className="pointer-events-none invisible absolute top-0 left-0 flex w-max items-center gap-x-2 px-0.5"
      >
        {items(false)}
      </ol>
    </div>
  );
}
