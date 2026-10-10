import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-x-1 rounded-xs border border-transparent font-medium",
  {
    variants: {
      size: {
        xs: "px-1.5 py-0.5 text-xs",
        sm: "h-7 px-2 text-sm",
      },
      variant: {
        default: "bg-control-bg text-control",
        secondary: "bg-accent/10 text-accent",
        destructive: "bg-error/10 text-error",
        warning: "bg-warning/10 text-warning",
        success: "bg-success/10 text-success",
      },
    },
    defaultVariants: {
      size: "xs",
      variant: "default",
    },
  }
);

type BadgeProps = ComponentProps<"span"> & VariantProps<typeof badgeVariants>;

function Badge({ className, size, variant, ...props }: BadgeProps) {
  return (
    <span
      className={cn(badgeVariants({ size, variant, className }))}
      {...props}
    />
  );
}

export type { BadgeProps };
export { Badge, badgeVariants };
