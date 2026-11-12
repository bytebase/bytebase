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
        "items-start rounded-sm border border-control-border p-3 transition-colors hover:bg-control-bg has-[[data-checked]]:border-accent has-[[data-checked]]:bg-accent/5",
        className
      )}
      radioClassName="mt-0.5"
      contentClassName="min-w-0 flex-1"
    >
      <span className="flex flex-col gap-1">
        <span className="font-medium text-main">{title}</span>
        <span className="text-xs leading-4 text-control-light">
          {description}
        </span>
      </span>
    </RadioGroupItem>
  );
}
