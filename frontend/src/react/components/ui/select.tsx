import { Select as BaseSelect } from "@base-ui/react/select";
import { Check, ChevronDown } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

// ---- Root ----
const Select = BaseSelect.Root;

// ---- Trigger ----
function SelectTrigger({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseSelect.Trigger>) {
  return (
    <BaseSelect.Trigger
      ref={ref}
      className={cn(
        "inline-flex items-center justify-between gap-1 h-8 px-2 text-sm rounded-md border border-control-border bg-white text-control whitespace-nowrap",
        "hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent",
        "disabled:pointer-events-none disabled:opacity-50",
        className
      )}
      {...props}
    >
      {children}
      <BaseSelect.Icon>
        <ChevronDown className="w-3.5 h-3.5 opacity-50 shrink-0" />
      </BaseSelect.Icon>
    </BaseSelect.Trigger>
  );
}

// ---- Value ----
const SelectValue = BaseSelect.Value;

// ---- Portal + Positioner + Popup  ----
function SelectContent({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseSelect.Popup>) {
  return (
    <BaseSelect.Portal>
      <BaseSelect.Positioner sideOffset={4}>
        <BaseSelect.Popup
          ref={ref}
          className={cn(
            "z-50 min-w-[var(--anchor-width)] max-h-60 overflow-auto rounded-md border border-control-border bg-white py-1 shadow-md",
            className
          )}
          {...props}
        >
          {children}
        </BaseSelect.Popup>
      </BaseSelect.Positioner>
    </BaseSelect.Portal>
  );
}

// ---- Item ----
function SelectItem({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseSelect.Item>) {
  return (
    <BaseSelect.Item
      ref={ref}
      className={cn(
        "relative flex items-center gap-2 px-2 py-1.5 pl-7 text-sm cursor-pointer select-none",
        "hover:bg-control-bg focus:bg-control-bg outline-hidden",
        "data-[highlighted]:bg-control-bg",
        className
      )}
      {...props}
    >
      <BaseSelect.ItemIndicator className="absolute left-1.5">
        <Check className="w-3.5 h-3.5" />
      </BaseSelect.ItemIndicator>
      <BaseSelect.ItemText>{children}</BaseSelect.ItemText>
    </BaseSelect.Item>
  );
}

export { Select, SelectTrigger, SelectValue, SelectContent, SelectItem };
