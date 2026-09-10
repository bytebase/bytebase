import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
import * as stylex from "@stylexjs/stylex";
import type { ReactNode } from "react";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { type ControlSize, controlMinHeightStyle } from "./styles.stylex";

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
  "aria-describedby"?: string;
  appearance?: "solid" | "soft";
  disabled?: boolean;
  className?: string;
  /** Segment size — matches the shared control size tier names. Defaults to `md`. */
  size?: ControlSize;
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onValueChange,
  ariaLabel,
  "aria-describedby": ariaDescribedBy,
  appearance = "solid",
  disabled = false,
  className,
  size = "md",
}: SegmentedControlProps<T>) {
  return (
    <BaseRadioGroup
      value={value}
      onValueChange={(nextValue) => {
        onValueChange(nextValue as T);
      }}
      disabled={disabled}
      aria-label={ariaLabel}
      aria-describedby={ariaDescribedBy}
      className={cn(
        "inline-flex max-w-full flex-wrap gap-px self-start rounded-xs border border-control-border bg-control-border",
        className
      )}
    >
      {options.map((option) => {
        const selected = option.value === value;
        const optionDisabled = disabled || option.disabled;
        const stylexProps = stylex.props(controlMinHeightStyle(size));
        const segment = (
          <label
            key={option.value}
            className={cn(
              "relative inline-flex flex-auto items-center justify-center transition-colors focus-within:outline-hidden focus-within:ring-2 focus-within:ring-accent focus-within:ring-inset",
              stylexProps.className,
              selected
                ? appearance === "soft"
                  ? "bg-accent/10 text-accent"
                  : "bg-accent text-accent-text"
                : "bg-background text-control hover:bg-control-bg",
              optionDisabled
                ? "cursor-not-allowed opacity-50 hover:bg-background"
                : "cursor-pointer"
            )}
            style={stylexProps.style}
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
