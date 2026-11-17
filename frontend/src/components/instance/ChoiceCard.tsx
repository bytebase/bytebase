import type { ComponentProps, ReactNode } from "react";
import { RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";

// A radio option styled as a tile, optionally showing a description beside its
// name for a choice the rest of the form depends on.
export function ChoiceCard({
  title,
  description,
  className,
  ...props
}: Omit<ComponentProps<typeof RadioGroupItem>, "children" | "title"> & {
  title: ReactNode;
  description?: ReactNode;
}) {
  const hasDescription = description !== undefined;
  return (
    <RadioGroupItem
      {...props}
      className={cn(
        "rounded-sm border border-control-border px-3 select-none transition-[color,background-color,border-color,transform] duration-150 ease-out motion-reduce:transition-none has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:border-accent/50 has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:bg-control-bg has-[>[role=radio]:not([data-disabled])]:active:scale-[0.99] motion-reduce:active:scale-100 has-[[data-checked]]:border-accent has-[[data-checked]]:bg-accent/5 has-[>[data-disabled]]:opacity-60",
        hasDescription ? "items-start py-2.5" : "min-h-9 items-center py-1.5",
        className
      )}
      radioClassName={hasDescription ? "mt-0.5" : undefined}
      contentClassName="min-w-0 flex-1"
    >
      {hasDescription ? (
        <span className="flex flex-col gap-1">
          <span className="font-medium text-main">{title}</span>
          <span className="text-xs leading-4 text-pretty text-control-light">
            {description}
          </span>
        </span>
      ) : (
        <span className="block leading-5 font-medium text-main">{title}</span>
      )}
    </RadioGroupItem>
  );
}
