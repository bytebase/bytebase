import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-3 py-0.5 text-sm font-medium",
  {
    variants: {
      variant: {
        default: "bg-control-bg text-control",
        secondary: "bg-accent/10 text-accent",
        destructive: "bg-error/10 text-error",
        warning: "bg-warning/10 text-warning",
        success: "bg-success/10 text-success",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

type BadgeProps = ComponentProps<"span"> & VariantProps<typeof badgeVariants>;

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant, className }))} {...props} />
  );
}

export type { BadgeProps };
export { Badge, badgeVariants };
