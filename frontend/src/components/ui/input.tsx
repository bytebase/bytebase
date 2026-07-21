import { Input as BaseInput } from "@base-ui/react/input";
import * as stylex from "@stylexjs/stylex";
import { cva } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";
import { type ControlSize, controlSizeStyle } from "./styles.stylex";

const inputVariants = cva(
  cn(
    "flex w-full rounded-xs border border-control-border bg-transparent text-main transition-colors",
    "placeholder:text-control-placeholder",
    "focus:outline-hidden",
    "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50",
    "read-only:cursor-default read-only:bg-control-bg read-only:focus:ring-0 read-only:focus:border-control-border"
  )
);

type InputProps = Omit<ComponentProps<"input">, "size"> & {
  size?: ControlSize;
};

function Input({ className, size = "md", ref, style, ...props }: InputProps) {
  const stylexProps = stylex.props(controlSizeStyle(size));
  return (
    <BaseInput
      {...props}
      ref={ref}
      className={cn(inputVariants(), stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
    />
  );
}

export type { InputProps };
export { Input };
