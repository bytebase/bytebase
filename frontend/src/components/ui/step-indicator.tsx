import { Check } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

export type StepIndicatorStep = {
  title: ReactNode;
  // Shown under the title in the vertical orientation only.
  description?: ReactNode;
  key?: string | number;
};

export function StepIndicator({
  steps,
  currentIndex,
  currentKey,
  className,
  fitOneRow = false,
  orientation = "horizontal",
  variant = "default",
  onStepClick,
  ariaLabel,
}: {
  readonly steps: StepIndicatorStep[];
  readonly currentIndex?: number;
  readonly currentKey?: string | number;
  readonly className?: string;
  // When the titles do not fit one row, only the current step keeps its title
  // in sight; the others stay for screen readers.
  readonly fitOneRow?: boolean;
  // Vertical is a rail of titles and descriptions beside a wizard's body.
  readonly orientation?: "horizontal" | "vertical";
  // "rail" draws the vertical rail's circles in one row: finished steps in
  // success, lines that fill the width, and only the current step's title in
  // sight. The vertical orientation always uses the rail look.
  readonly variant?: "default" | "rail";
  // Makes each step a button that moves the wizard to it.
  readonly onStepClick?: (index: number) => void;
  readonly ariaLabel?: string;
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

  const stepButton = (index: number, body: ReactNode) =>
    onStepClick ? (
      <button
        type="button"
        aria-current={index === activeIndex ? "step" : undefined}
        onClick={() => onStepClick(index)}
        className="flex items-start gap-x-3 rounded-xs text-left focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
      >
        {body}
      </button>
    ) : (
      <div
        aria-current={index === activeIndex ? "step" : undefined}
        className="flex items-start gap-x-3"
      >
        {body}
      </div>
    );

  const railCircle = (index: number) => {
    const isCurrent = index === activeIndex;
    const isCompleted = activeIndex >= 0 && index < activeIndex;
    return (
      <div
        className={cn(
          "flex size-7 shrink-0 items-center justify-center rounded-full border text-xs font-medium",
          isCompleted && "border-success bg-success text-accent-text",
          isCurrent && "border-accent bg-accent text-accent-text",
          !isCompleted &&
            !isCurrent &&
            "border-control-border bg-background text-control-light"
        )}
      >
        {isCompleted ? <Check className="size-4" /> : index + 1}
      </div>
    );
  };

  if (orientation === "horizontal" && variant === "rail") {
    return (
      <ol
        aria-label={ariaLabel}
        data-slot="step-indicator"
        data-variant="rail"
        className={cn("flex w-full items-center", className)}
      >
        {steps.map((step, index) => {
          const isCurrent = index === activeIndex;
          const isCompleted = activeIndex >= 0 && index < activeIndex;
          const isLast = index === steps.length - 1;
          return (
            <li
              key={step.key ?? index}
              // flex-auto, not flex-1: a zero basis would squeeze the current
              // step's title over its line.
              className={cn(
                "flex items-center gap-x-2",
                !isLast && "flex-auto"
              )}
            >
              {stepButton(
                index,
                <>
                  {railCircle(index)}
                  <span
                    className={cn(
                      "self-center text-sm font-medium whitespace-nowrap text-main",
                      !isCurrent && "sr-only"
                    )}
                  >
                    {step.title}
                  </span>
                </>
              )}
              {!isLast && (
                <div
                  aria-hidden="true"
                  className={cn(
                    "mr-2 h-0.5 min-w-3 flex-1",
                    isCompleted ? "bg-success" : "bg-control-border"
                  )}
                />
              )}
            </li>
          );
        })}
      </ol>
    );
  }

  if (orientation === "vertical") {
    return (
      <ol
        aria-label={ariaLabel}
        data-slot="step-indicator"
        data-orientation="vertical"
        className={cn("flex flex-col", className)}
      >
        {steps.map((step, index) => {
          const isCurrent = index === activeIndex;
          const isCompleted = activeIndex >= 0 && index < activeIndex;
          const isLast = index === steps.length - 1;
          return (
            <li key={step.key ?? index} className="relative flex pb-6">
              {!isLast && (
                <div
                  aria-hidden="true"
                  className={cn(
                    "absolute top-7 bottom-0 left-3.5 w-0.5 -translate-x-1/2",
                    isCompleted ? "bg-success" : "bg-control-border"
                  )}
                />
              )}
              {stepButton(
                index,
                <>
                  {railCircle(index)}
                  <div className="flex min-w-0 flex-col pt-0.5">
                    <span
                      className={cn(
                        "text-sm font-medium",
                        isCompleted || isCurrent
                          ? "text-main"
                          : "text-control-light"
                      )}
                    >
                      {step.title}
                    </span>
                    {step.description !== undefined && (
                      <span className="text-xs text-control-light">
                        {step.description}
                      </span>
                    )}
                  </div>
                </>
              )}
            </li>
          );
        })}
      </ol>
    );
  }

  const items = (compact: boolean) =>
    steps.map((step, index) => {
      const isReached = activeIndex >= 0 && index <= activeIndex;
      const isCompleted = activeIndex >= 0 && index < activeIndex;
      const body = (
        <div className="flex items-center gap-x-2">
          <div
            className={cn(
              "flex size-6 items-center justify-center rounded-full text-xs font-medium",
              isReached
                ? "bg-accent text-accent-text"
                : "bg-control-bg-hover text-control-light"
            )}
          >
            {isCompleted ? (
              <Check
                className="size-3.5"
                role="img"
                aria-label={String(index + 1)}
              />
            ) : (
              index + 1
            )}
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
      );
      return (
        <li
          key={step.key ?? index}
          aria-current={
            !onStepClick && index === activeIndex ? "step" : undefined
          }
          className="flex items-center gap-x-2"
        >
          {index > 0 && (
            <div
              className={cn("h-px bg-control-border", compact ? "w-4" : "w-8")}
            />
          )}
          {onStepClick ? (
            <button
              type="button"
              aria-current={index === activeIndex ? "step" : undefined}
              onClick={() => onStepClick(index)}
              className="rounded-xs focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
            >
              {body}
            </button>
          ) : (
            body
          )}
        </li>
      );
    });

  const list = (
    <ol
      ref={listRef}
      aria-label={ariaLabel}
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
