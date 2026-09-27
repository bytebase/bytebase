import { Toast as BaseToast } from "@base-ui/react/toast";
import { cva } from "class-variance-authority";
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/utils";

// Map BBNotificationStyle ("SUCCESS" | "INFO" | "WARN" | "CRITICAL") onto
// the visual variant of the toast container.
export type ToastVariant = "success" | "info" | "warning" | "error";

// Toast card: neutral border (variant is signalled by the icon color, matching
// sonner / react-hot-toast / shadcn-default — a full colored border is visual
// noise when the icon already carries the type signal).
const toastRoot = [
  "absolute right-0 bottom-0",
  "w-(--toast-width) max-w-[calc(100vw-2rem)]",
  "rounded-sm border bg-background text-main shadow-md",
  "px-4 py-3 pr-10 overflow-hidden origin-top",
  // Index 0 is newest. Keep paint order independent of DOM insertion order.
  "[z-index:calc(100_-_var(--toast-index))]",
  // Collapsed cards share the front card's height so every sheet edge shows.
  "h-(--toast-frontmost-height) [&[data-expanded]]:h-(--toast-height)",
  // Base UI emits these CSS vars; we use them for the stack/expand transforms.
  "[transition:transform_250ms,opacity_250ms,height_250ms]",
  "[transform:translateY(calc(var(--toast-index)_*_-12px))_scale(calc(1_-_var(--toast-index)_*_0.05))]",
  "[&[data-expanded]]:[transform:translateY(calc(var(--toast-offset-y,0px)_*_-1_-_var(--toast-index)_*_16px))]",
  "[&[data-limited]]:opacity-0 [&[data-limited]]:pointer-events-none",
  "[&[data-starting-style]]:opacity-0",
  "[&[data-ending-style]]:opacity-0",
].join(" ");

const iconVariants = cva("size-5 shrink-0 mt-0.5", {
  variants: {
    variant: {
      success: "text-success",
      info: "text-info",
      warning: "text-warning",
      error: "text-error",
    },
  },
  defaultVariants: { variant: "info" },
});

const iconMap: Record<ToastVariant, typeof CheckCircle2> = {
  success: CheckCircle2,
  info: Info,
  warning: AlertTriangle,
  error: XCircle,
};

type ToastRootProps = Omit<
  ComponentProps<typeof BaseToast.Root>,
  "className"
> & {
  variant?: ToastVariant;
  className?: string;
  showIcon?: boolean;
  children?: ReactNode;
};

function ToastRoot({
  variant = "info",
  showIcon = true,
  className,
  children,
  ...props
}: ToastRootProps) {
  const Icon = iconMap[variant];
  return (
    <BaseToast.Root {...props} className={cn(toastRoot, className)}>
      <BaseToast.Content className="flex items-start gap-x-3 [&[data-behind]:not([data-expanded])]:opacity-0">
        {showIcon ? <Icon className={iconVariants({ variant })} /> : null}
        <div className="flex min-w-0 flex-1 flex-col gap-y-1">{children}</div>
      </BaseToast.Content>
    </BaseToast.Root>
  );
}

function ToastTitle({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Title>) {
  return (
    <BaseToast.Title
      {...props}
      className={cn("text-sm font-medium leading-5", className)}
    />
  );
}

function ToastDescription({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Description>) {
  return (
    <BaseToast.Description
      {...props}
      className={cn(
        "text-sm leading-5 text-control-light whitespace-pre-wrap",
        className
      )}
    />
  );
}

function ToastAction({
  className,
  ...props
}: ComponentProps<typeof BaseToast.Action>) {
  return (
    <BaseToast.Action
      {...props}
      className={cn(
        "mt-1 inline-flex w-fit text-sm font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
        className
      )}
    />
  );
}

function ToastClose({
  className,
  "aria-label": ariaLabel,
  ...props
}: ComponentProps<typeof BaseToast.Close>) {
  return (
    <BaseToast.Close
      {...props}
      aria-label={ariaLabel ?? "Close"}
      className={cn(
        "absolute right-2 top-2 inline-flex size-7 items-center justify-center rounded-sm text-control-light opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
        className
      )}
    >
      <X className="size-4" />
    </BaseToast.Close>
  );
}

export { ToastAction, ToastClose, ToastDescription, ToastRoot, ToastTitle };
