import { Input as BaseInput } from "@base-ui/react/input";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

type InputProps = ComponentProps<"input">;

function Input({ className, ref, ...props }: InputProps) {
  return (
    <BaseInput
      ref={ref}
      className={cn(
        "flex h-9 w-full rounded-md border border-control-border bg-transparent px-3 py-1 text-sm text-main transition-colors",
        "placeholder:text-control-placeholder",
        "focus:outline-hidden focus:ring-2 focus:ring-accent focus:border-accent",
        "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50",
        "read-only:cursor-default read-only:bg-control-bg read-only:focus:ring-0 read-only:focus:border-control-border",
        className
      )}
      {...props}
    />
  );
}

export { Input };
export type { InputProps };
