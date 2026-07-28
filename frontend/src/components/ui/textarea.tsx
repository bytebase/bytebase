import * as stylex from "@stylexjs/stylex";
import { cva } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";
import { type ControlSize, controlMultilineSizeStyle } from "./styles.stylex";

const textareaVariants = cva(
  cn(
    "flex min-h-25 w-full rounded-xs border border-control-border bg-transparent text-main transition-colors",
    "placeholder:text-control-placeholder",
    "focus:outline-hidden focus:ring-1 focus:ring-accent focus:border-accent",
    "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50"
  )
);

type TextareaProps = Omit<ComponentProps<"textarea">, "size"> & {
  size?: ControlSize;
};

function Textarea({
  className,
  size = "md",
  ref,
  style,
  ...props
}: TextareaProps) {
  const stylexProps = stylex.props(controlMultilineSizeStyle(size));
  return (
    <textarea
      {...props}
      ref={ref}
      className={cn(textareaVariants(), stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
    />
  );
}

export type { TextareaProps };
export { Textarea };
