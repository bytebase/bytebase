import { cva, type VariantProps } from "class-variance-authority";
import {
  AlertCircle,
  AlertTriangle,
  Info,
  type LucideIcon,
} from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const alertVariants = cva(
  "relative w-full rounded-xs border px-4 py-3 text-sm flex gap-x-3 items-center",
  {
    variants: {
      variant: {
        info: "border-accent/30 bg-accent/5 text-accent",
        warning: "border-warning/30 bg-warning/5 text-warning",
        error: "border-error/30 bg-error/5 text-error",
      },
    },
    defaultVariants: {
      variant: "info",
    },
  }
);

const iconMap: Record<string, LucideIcon> = {
  info: Info,
  warning: AlertTriangle,
  error: AlertCircle,
};

type AlertProps = ComponentProps<"div"> &
  VariantProps<typeof alertVariants> & {
    showIcon?: boolean;
  };

function Alert({
  className,
  variant = "info",
  showIcon = true,
  children,
  ...props
}: AlertProps) {
  const Icon = iconMap[variant ?? "info"];
  return (
    <div
      role="alert"
      className={cn(alertVariants({ variant, className }))}
      {...props}
    >
      {showIcon && <Icon className="h-5 w-5 shrink-0 mt-0.5" />}
      <div>{children}</div>
    </div>
  );
}

function AlertTitle({ className, ...props }: ComponentProps<"h5">) {
  return (
    <h5 className={cn("font-medium leading-tight", className)} {...props} />
  );
}

function AlertDescription({ className, ...props }: ComponentProps<"div">) {
  return (
    <div className={cn("mt-1 text-sm opacity-90", className)} {...props} />
  );
}

export { Alert, AlertTitle, AlertDescription, alertVariants };
export type { AlertProps };
