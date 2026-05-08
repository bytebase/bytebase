import { Input as BaseInput } from "@base-ui/react/input";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const inputVariants = cva(
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

type InputProps = Omit<ComponentProps<"input">, "size"> &
  VariantProps<typeof inputVariants>;

function Input({ className, size, ref, ...props }: InputProps) {
  return (
    <BaseInput
      ref={ref}
      className={cn(inputVariants({ size }), className)}
      {...props}
    />
  );
}

export { Input };
export type { InputProps };
