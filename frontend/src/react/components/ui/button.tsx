import { Button as BaseButton } from "@base-ui/react/button";
import * as stylex from "@stylexjs/stylex";
import { cva, type VariantProps } from "class-variance-authority";
import type { ClassValue } from "clsx";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import {
  buttonGapStyle,
  type ControlSize,
  controlSizeStyle,
} from "./styles.stylex";

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
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

type ButtonVariantProps = VariantProps<typeof buttonVariantClasses> & {
  class?: ClassValue;
  className?: ClassValue;
  size?: ControlSize | "default";
};

function buttonVariants({
  class: classValue,
  className,
  size = "default",
  variant,
}: ButtonVariantProps = {}) {
  const resolvedSize = size === "default" ? "md" : size;
  const stylexProps = stylex.props(
    controlSizeStyle(resolvedSize),
    buttonGapStyle(resolvedSize)
  );
  return cn(
    buttonVariantClasses({ variant }),
    stylexProps.className,
    classValue,
    className
  );
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
