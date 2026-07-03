import { Button as BaseButton } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";
import type { ClassValue } from "clsx";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const buttonVariantClasses = cva(
  "inline-flex items-center justify-center rounded-xs font-medium whitespace-nowrap cursor-pointer transition-colors focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-accent text-accent-text hover:bg-accent-hover",
        outline:
          "border border-control-border bg-transparent hover:bg-control-bg text-control",
        ghost: "hover:bg-control-bg text-control",
        destructive: "bg-error text-white hover:bg-error-hover",
        "ghost-destructive":
          "border border-control-border border-error text-error hover:bg-error hover:text-white",
        link: "text-accent underline-offset-4 hover:underline",
      },
      size: {
        // `default` is an alias for `md` — both render identically.
        default: "h-9 px-3 text-sm leading-5 gap-1.5",
        xs: "h-6 px-1.5 text-xs leading-4 gap-1",
        sm: "h-7 px-2 text-xs leading-4 gap-1",
        md: "h-9 px-3 text-sm leading-5 gap-1.5",
        lg: "h-10 px-4 text-sm leading-5 gap-1.5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

type ButtonVariantProps = VariantProps<typeof buttonVariantClasses> & {
  class?: ClassValue;
  className?: ClassValue;
};

function buttonVariants({
  class: classValue,
  className,
  size = "default",
  variant,
}: ButtonVariantProps = {}) {
  return cn(buttonVariantClasses({ size, variant }), classValue, className);
}

type ButtonProps = ComponentProps<"button"> & ButtonVariantProps;

function Button({
  class: classValue,
  className,
  variant,
  size = "default",
  ref,
  style,
  ...props
}: ButtonProps) {
  return (
    <BaseButton
      {...props}
      ref={ref}
      className={buttonVariants({
        class: classValue,
        className,
        size,
        variant,
      })}
      style={style}
    />
  );
}

export type { ButtonProps };
export { Button, buttonVariants };
