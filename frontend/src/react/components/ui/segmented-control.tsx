import type { ReactNode } from "react";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";

export interface SegmentedControlOption<T extends string> {
  value: T;
  label: ReactNode;
  disabled?: boolean;
  tooltip?: ReactNode;
}

interface SegmentedControlProps<T extends string> {
  value: T;
  options: SegmentedControlOption<T>[];
  onValueChange: (value: T) => void;
  ariaLabel: string;
  disabled?: boolean;
  className?: string;
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onValueChange,
  ariaLabel,
  disabled = false,
  className,
}: SegmentedControlProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={cn(
        "inline-flex max-w-full flex-wrap rounded-xs border border-control-border bg-background",
        className
      )}
    >
      {options.map((option, index) => {
        const selected = option.value === value;
        const optionDisabled = disabled || option.disabled;
        const button = (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-disabled={optionDisabled || undefined}
            data-state={selected ? "checked" : "unchecked"}
            data-disabled={optionDisabled || undefined}
            className={cn(
              "min-h-8 px-3 text-sm transition-colors focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
              index > 0 && "border-l border-control-border",
              selected
                ? "bg-accent text-accent-text"
                : "bg-background text-control hover:bg-control-bg",
              optionDisabled &&
                "cursor-not-allowed opacity-50 hover:bg-background"
            )}
            onClick={() => {
              if (!optionDisabled) {
                onValueChange(option.value);
              }
            }}
          >
            {option.label}
          </button>
        );

        if (!option.tooltip) {
          return button;
        }

        return (
          <Tooltip key={option.value} content={option.tooltip}>
            {button}
          </Tooltip>
        );
      })}
    </div>
  );
}
