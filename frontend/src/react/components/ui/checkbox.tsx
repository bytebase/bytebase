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

interface CheckboxProps
  extends Omit<
    React.ComponentProps<typeof BaseCheckbox.Root>,
    "checked" | "onCheckedChange" | "className" | "children"
  > {
  checked: boolean | "indeterminate";
  onCheckedChange?: (checked: boolean) => void;
  size?: CheckboxSize;
  className?: string;
}

function Checkbox({
  checked,
  onCheckedChange,
  onClick,
  size = "md",
  className,
  ...rootProps
}: CheckboxProps) {
  const baseChecked = checked === "indeterminate" ? false : checked;
  const indeterminate = checked === "indeterminate";

  const root = (
    <BaseCheckbox.Root
      {...rootProps}
      checked={baseChecked}
      indeterminate={indeterminate}
      onCheckedChange={(value) => onCheckedChange?.(value)}
      className={cn(
        "inline-flex shrink-0 items-center justify-center align-middle rounded-sm border bg-background transition-colors",
        ROOT_SIZE[size],
        "border-control-border hover:border-accent/60",
        "data-[checked]:bg-accent data-[checked]:border-accent",
        "data-[indeterminate]:bg-accent data-[indeterminate]:border-accent",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "data-disabled:cursor-not-allowed data-disabled:hover:border-control-border",
        // Disabled + checked: render in a muted gray instead of the
        // accent so it reads as "locked-on". Base UI sets the
        // `data-disabled` attribute when `disabled` is true — using
        // `data-disabled:` instead of `disabled:` matches the
        // convention used elsewhere in this directory and is
        // independent of whether the rendered element supports the
        // `:disabled` pseudo-class.
        "data-disabled:data-[checked]:bg-control-light data-disabled:data-[checked]:border-control-light",
        "data-disabled:data-[indeterminate]:bg-control-light data-disabled:data-[indeterminate]:border-control-light",
        "data-disabled:not-data-[checked]:not-data-[indeterminate]:opacity-50",
        !onClick && className
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

  if (!onClick) return root;

  return (
    <span
      className={cn("inline-flex align-middle", className)}
      onClick={onClick}
    >
      {root}
    </span>
  );
}

export type { CheckboxProps, CheckboxSize };
export { Checkbox };
