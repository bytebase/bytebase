import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { Check, Minus } from "lucide-react";
import type * as React from "react";
import { cn } from "@/react/lib/utils";

type CheckboxSize = "sm" | "md";

const ROOT_SIZE: Record<CheckboxSize, string> = {
  sm: "size-3.5",
  md: "size-4",
};

const ICON_SIZE: Record<CheckboxSize, string> = {
  sm: "size-2.5",
  md: "size-3",
};

interface CheckboxProps {
  checked: boolean | "indeterminate";
  onCheckedChange?: (checked: boolean) => void;
  onClick?: React.MouseEventHandler<HTMLElement>;
  disabled?: boolean;
  size?: CheckboxSize;
  className?: string;
  id?: string;
  name?: string;
  "aria-label"?: string;
}

function Checkbox({
  checked,
  onCheckedChange,
  onClick,
  disabled,
  size = "md",
  className,
  id,
  name,
  "aria-label": ariaLabel,
}: CheckboxProps) {
  const baseChecked = checked === "indeterminate" ? false : checked;
  const indeterminate = checked === "indeterminate";

  return (
    <BaseCheckbox.Root
      checked={baseChecked}
      indeterminate={indeterminate}
      onCheckedChange={(value) => onCheckedChange?.(value)}
      onClick={onClick}
      disabled={disabled}
      id={id}
      name={name}
      aria-label={ariaLabel}
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-sm border bg-background transition-colors",
        ROOT_SIZE[size],
        "border-control-border hover:border-accent/60",
        "data-[checked]:bg-accent data-[checked]:border-accent",
        "data-[indeterminate]:bg-accent data-[indeterminate]:border-accent",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-control-border",
        className
      )}
    >
      <BaseCheckbox.Indicator className="flex items-center justify-center text-background">
        {indeterminate ? (
          <Minus className={ICON_SIZE[size]} />
        ) : (
          <Check className={ICON_SIZE[size]} />
        )}
      </BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
  );
}

export { Checkbox };
export type { CheckboxProps, CheckboxSize };
