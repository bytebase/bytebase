import { cva, type VariantProps } from "class-variance-authority";
import {
  AlertCircle,
  AlertTriangle,
  Info,
  type LucideIcon,
} from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";

const alertVariants = cva(
  "relative flex w-full items-start gap-x-3 rounded-xs border px-4 py-3 text-sm leading-5 text-control shadow-xs",
  {
    variants: {
      variant: {
        info: "border-info/40 bg-info/10",
        warning: "border-warning/40 bg-warning/10",
        error: "border-error/40 bg-error/10",
      },
    },
    defaultVariants: {
      variant: "info",
    },
  }
);

const alertIconVariants = cva("mt-0.5 size-5 shrink-0", {
  variants: {
    variant: {
      info: "text-info",
      warning: "text-warning",
      error: "text-error",
    },
  },
  defaultVariants: {
    variant: "info",
  },
});

const iconMap: Record<string, LucideIcon> = {
  info: Info,
  warning: AlertTriangle,
  error: AlertCircle,
};

type AlertProps = Omit<ComponentProps<"div">, "title"> &
  VariantProps<typeof alertVariants> & {
    title?: ReactNode;
    description?: ReactNode;
    showIcon?: boolean;
  };

function Alert({
  className,
  variant = "info",
  showIcon = true,
  title,
  description,
  children,
  ...props
}: AlertProps) {
  const Icon = iconMap[variant ?? "info"];
  const hasStructuredContent = title !== undefined || description !== undefined;

  return (
    <div
      role="alert"
      className={cn(alertVariants({ variant, className }))}
      {...props}
    >
      {showIcon && <Icon className={alertIconVariants({ variant })} />}
      <div className="min-w-0 flex-1">
        {hasStructuredContent ? (
          <>
            {title !== undefined && (
              <h5 className="font-medium leading-6">{title}</h5>
            )}
            {description !== undefined && (
              <div
                className={cn(
                  "text-sm text-control-light leading-6",
                  title !== undefined && "mt-1"
                )}
              >
                {description}
              </div>
            )}
            {children}
          </>
        ) : (
          children
        )}
      </div>
    </div>
  );
}

export type { AlertProps };
export { Alert, alertVariants };
