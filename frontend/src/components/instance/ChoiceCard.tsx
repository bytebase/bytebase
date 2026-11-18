import type { ComponentProps, ReactNode } from "react";
import { RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";

// A radio option that shows its consequence beside its name, for a choice the
// rest of the form depends on.
export function ChoiceCard({
  title,
  description,
  className,
  ...props
}: Omit<ComponentProps<typeof RadioGroupItem>, "children" | "title"> & {
  title: ReactNode;
  description: ReactNode;
}) {
  return (
    <RadioGroupItem
      {...props}
      className={cn(
        "items-start rounded-sm border border-control-border px-3 py-2.5 select-none transition-[color,background-color,border-color,box-shadow,transform] duration-150 ease-out motion-reduce:transition-none has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:border-accent/50 has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:bg-control-bg has-[>[role=radio]:not([data-disabled])]:active:scale-[0.99] motion-reduce:active:scale-100 has-[[data-checked]]:border-accent has-[[data-checked]]:bg-accent/5 has-[[data-checked]]:ring-1 has-[[data-checked]]:ring-inset has-[[data-checked]]:ring-accent has-[>[data-disabled]]:opacity-60",
        className
      )}
      radioClassName="mt-0.5"
      contentClassName="min-w-0 flex-1"
    >
      <span className="flex flex-col gap-1">
        <span className="font-medium text-main">{title}</span>
        <span className="text-xs leading-4 text-pretty text-control-light">
          {description}
        </span>
      </span>
    </RadioGroupItem>
  );
}
