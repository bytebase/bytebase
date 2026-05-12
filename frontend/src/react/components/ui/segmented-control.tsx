import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
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
  /** Segment size — matches the Input/Combobox size tier names. Defaults to `md`. */
  size?: "sm" | "md";
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onValueChange,
  ariaLabel,
  disabled = false,
  className,
  size = "md",
}: SegmentedControlProps<T>) {
  const segmentSizeClasses =
    size === "sm" ? "min-h-7 px-2 text-xs" : "min-h-8 px-3 text-sm";
  return (
    <BaseRadioGroup
      value={value}
      onValueChange={(nextValue) => {
        onValueChange(nextValue as T);
      }}
      disabled={disabled}
      aria-label={ariaLabel}
      className={cn(
        "inline-flex max-w-full flex-wrap self-start rounded-xs border border-control-border bg-background",
        className
      )}
    >
      {options.map((option, index) => {
        const selected = option.value === value;
        const previousSelected =
          index > 0 && options[index - 1]?.value === value;
        const optionDisabled = disabled || option.disabled;
        const segment = (
          <label
            key={option.value}
            className={cn(
              "relative inline-flex items-center justify-center transition-colors focus-within:outline-hidden focus-within:ring-2 focus-within:ring-accent focus-within:ring-inset",
              segmentSizeClasses,
              index > 0 &&
                !previousSelected &&
                "border-l border-control-border",
              selected
                ? "bg-accent text-accent-text"
                : "bg-background text-control hover:bg-control-bg",
              optionDisabled &&
                "cursor-not-allowed opacity-50 hover:bg-background"
            )}
          >
            <Radio.Root
              value={option.value}
              disabled={optionDisabled}
              aria-checked={selected}
              aria-disabled={optionDisabled || undefined}
              data-state={selected ? "checked" : "unchecked"}
              data-disabled={optionDisabled || undefined}
              className="sr-only"
            />
            <span
              className={cn(
                "pointer-events-none select-none",
                optionDisabled && "cursor-not-allowed"
              )}
            >
              {option.label}
            </span>
          </label>
        );

        if (!option.tooltip) {
          return segment;
        }

        return (
          <Tooltip key={option.value} content={option.tooltip}>
            {segment}
          </Tooltip>
        );
      })}
    </BaseRadioGroup>
  );
}
