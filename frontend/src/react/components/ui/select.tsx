import { Select as BaseSelect } from "@base-ui/react/select";
import { cva, type VariantProps } from "class-variance-authority";
import { Check, ChevronDown } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

// ---- Root ----
const Select = BaseSelect.Root;

// ---- Trigger ----
const selectTriggerVariants = cva(
  cn(
    "inline-flex items-center justify-between gap-1 rounded-xs border border-control-border bg-background text-control whitespace-nowrap",
    "hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent",
    "disabled:pointer-events-none disabled:opacity-50"
  ),
  {
    variants: {
      size: {
        xs: "h-6 px-2 text-xs leading-4",
        sm: "h-7 px-2 text-xs leading-4",
        md: "h-9 px-3 text-sm leading-5",
        lg: "h-10 px-4 text-sm leading-5",
      },
    },
    defaultVariants: {
      size: "md",
    },
  }
);

type SelectTriggerProps = ComponentProps<typeof BaseSelect.Trigger> &
  VariantProps<typeof selectTriggerVariants>;

function SelectTrigger({
  className,
  children,
  ref,
  size,
  ...props
}: SelectTriggerProps) {
  return (
    <BaseSelect.Trigger
      ref={ref}
      className={cn(selectTriggerVariants({ size }), className)}
      {...props}
    >
      {children}
      <BaseSelect.Icon>
        <ChevronDown className="size-3.5 opacity-50 shrink-0" />
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
    <BaseSelect.Portal container={getLayerRoot("overlay")}>
      <BaseSelect.Positioner sideOffset={4} className={LAYER_SURFACE_CLASS}>
        <BaseSelect.Popup
          ref={ref}
          className={cn(
            "min-w-(--anchor-width) max-h-60 overflow-auto rounded-sm border border-control-border bg-background py-1 shadow-md",
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
        "data-highlighted:bg-control-bg",
        className
      )}
      {...props}
    >
      <BaseSelect.ItemIndicator className="absolute left-1.5">
        <Check className="size-3.5" />
      </BaseSelect.ItemIndicator>
      <BaseSelect.ItemText>{children}</BaseSelect.ItemText>
    </BaseSelect.Item>
  );
}

export { Select, SelectTrigger, SelectValue, SelectContent, SelectItem };
