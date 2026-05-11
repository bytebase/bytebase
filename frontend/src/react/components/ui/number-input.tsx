import { NumberField } from "@base-ui/react/number-field";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { cn } from "@/react/lib/utils";

// Mirrors the sizing/appearance of ./input.tsx so NumberInput visually matches
// the regular Input at every size.
const numberInputClasses = cva(
  cn(
    "flex w-full rounded-xs border border-control-border bg-transparent text-main transition-colors",
    "placeholder:text-control-placeholder",
    "focus:outline-hidden",
    "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50",
    "read-only:cursor-default read-only:bg-control-bg read-only:focus:ring-0 read-only:focus:border-control-border"
  ),
  {
    variants: {
      size: {
        xs: "h-6 px-2 text-xs leading-4",
        sm: "h-7 px-2 text-xs leading-4",
        md: "h-9 px-3 text-sm leading-5",
        lg: "h-10 px-4 text-sm leading-5",
      },
    },
    defaultVariants: {
      size: "md",
    },
  }
);

type NumberFieldRootProps = ComponentPropsWithoutRef<typeof NumberField.Root>;

interface NumberInputProps
  extends Omit<
      NumberFieldRootProps,
      "className" | "render" | "size" | "prefix"
    >,
    VariantProps<typeof numberInputClasses> {
  /** Applied to the outer wrapper; use for layout/width classes (e.g. "w-60"). */
  className?: string;
  /** Applied to the underlying input element for extra input-specific styling. */
  inputClassName?: string;
  placeholder?: string;
  /** Content rendered inside the field to the right of the input (e.g. unit). */
  suffix?: ReactNode;
  /** Content rendered inside the field to the left of the input. */
  prefix?: ReactNode;
}

function NumberInput({
  className,
  inputClassName,
  size,
  suffix,
  prefix,
  placeholder,
  ...rootProps
}: NumberInputProps) {
  const hasAffix = Boolean(prefix || suffix);
  const inputClasses = cn(
    numberInputClasses({ size }),
    prefix && "pl-10",
    suffix && "pr-12",
    inputClassName
  );

  if (!hasAffix) {
    return (
      <NumberField.Root {...rootProps} className={className}>
        <NumberField.Input placeholder={placeholder} className={inputClasses} />
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
        <NumberField.Input placeholder={placeholder} className={inputClasses} />
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
