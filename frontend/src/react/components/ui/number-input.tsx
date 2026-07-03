import { NumberField } from "@base-ui/react/number-field";
import * as stylex from "@stylexjs/stylex";
import { cva } from "class-variance-authority";
import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { cn } from "@/react/lib/utils";
import { type ControlSize, controlSizeStyle } from "./styles.stylex";

// Mirrors the sizing/appearance of ./input.tsx so NumberInput visually matches
// the regular Input at every size.
const numberInputClasses = cva(
  cn(
    "flex w-full rounded-xs border border-control-border bg-transparent text-main transition-colors",
    "placeholder:text-control-placeholder",
    "focus:outline-hidden",
    "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50",
    "read-only:cursor-default read-only:bg-control-bg read-only:focus:ring-0 read-only:focus:border-control-border"
  )
);

const numberInputPaddingClasses = cva("", {
  variants: {
    size: {
      xs: "px-1.5",
      sm: "px-2",
      md: "px-3",
      lg: "px-4",
    },
  },
});

type NumberFieldRootProps = ComponentPropsWithoutRef<typeof NumberField.Root>;

interface NumberInputProps
  extends Omit<
    NumberFieldRootProps,
    "className" | "render" | "size" | "prefix"
  > {
  /** Applied to the outer wrapper; use for layout/width classes (e.g. "w-60"). */
  className?: string;
  /** Applied to the underlying input element for extra input-specific styling. */
  inputClassName?: string;
  placeholder?: string;
  size?: ControlSize;
  /** Content rendered inside the field to the right of the input (e.g. unit). */
  suffix?: ReactNode;
  /** Content rendered inside the field to the left of the input. */
  prefix?: ReactNode;
}

function NumberInput({
  className,
  inputClassName,
  size = "md",
  suffix,
  prefix,
  placeholder,
  ...rootProps
}: NumberInputProps) {
  const hasAffix = Boolean(prefix || suffix);
  const stylexProps = stylex.props(
    controlSizeStyle(size, { paddingInline: false })
  );
  const inputClasses = cn(
    numberInputClasses(),
    stylexProps.className,
    numberInputPaddingClasses({ size }),
    prefix && "pl-10",
    suffix && "pr-12",
    inputClassName
  );

  if (!hasAffix) {
    return (
      <NumberField.Root {...rootProps} className={className}>
        <NumberField.Input
          placeholder={placeholder}
          className={inputClasses}
          style={stylexProps.style}
        />
      </NumberField.Root>
    );
  }

  return (
    <NumberField.Root {...rootProps} className={className}>
      <NumberField.Group className="relative inline-flex w-full items-center">
        {prefix && (
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-control-light">
            {prefix}
          </span>
        )}
        <NumberField.Input
          placeholder={placeholder}
          className={inputClasses}
          style={stylexProps.style}
        />
        {suffix && (
          <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-control-light">
            {suffix}
          </span>
        )}
      </NumberField.Group>
    </NumberField.Root>
  );
}

export type { NumberInputProps };
export { NumberInput };
