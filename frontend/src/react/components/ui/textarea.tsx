import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const textareaVariants = cva(
  cn(
    "flex min-h-25 w-full rounded-xs border border-control-border bg-transparent text-main transition-colors",
    "placeholder:text-control-placeholder",
    "focus:outline-hidden focus:ring-1 focus:ring-accent focus:border-accent",
    "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50"
  ),
  {
    variants: {
      size: {
        xs: "px-2 py-1 text-xs leading-4",
        sm: "px-2 py-1.5 text-xs leading-4",
        md: "px-3 py-2 text-sm leading-5",
        lg: "px-4 py-2 text-sm leading-5",
      },
    },
    defaultVariants: {
      size: "md",
    },
  }
);

type TextareaProps = Omit<ComponentProps<"textarea">, "size"> &
  VariantProps<typeof textareaVariants>;

function Textarea({ className, size, ref, ...props }: TextareaProps) {
  return (
    <textarea
      ref={ref}
      className={cn(textareaVariants({ size }), className)}
      {...props}
    />
  );
}

export type { TextareaProps };
export { Textarea };
