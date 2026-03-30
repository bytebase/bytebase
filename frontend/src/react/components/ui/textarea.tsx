import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

type TextareaProps = ComponentProps<"textarea">;

function Textarea({ className, ref, ...props }: TextareaProps) {
  return (
    <textarea
      ref={ref}
      className={cn(
        "flex min-h-[100px] w-full rounded-md border border-control-border bg-transparent px-3 py-2 text-sm text-main transition-colors",
        "placeholder:text-control-placeholder",
        "focus:outline-hidden focus:ring-2 focus:ring-accent focus:border-accent",
        "disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50",
        className
      )}
      {...props}
    />
  );
}

export { Textarea };
export type { TextareaProps };
