import { Select as BaseSelect } from "@base-ui/react/select";
import * as stylex from "@stylexjs/stylex";
import { cva } from "class-variance-authority";
import { Check, ChevronDown } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";
import {
  type ControlSize,
  controlSizeStyle,
  menuRowStateClassName,
  menuRowStyle,
} from "./styles.stylex";

// ---- Root ----
const Select = BaseSelect.Root;

// ---- Trigger ----
const selectTriggerVariants = cva(
  cn(
    "inline-flex items-center justify-between gap-1 rounded-xs border border-control-border bg-background text-control whitespace-nowrap",
    "cursor-pointer",
    "hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent",
    "disabled:pointer-events-none disabled:opacity-50"
  )
);

type SelectTriggerProps = ComponentProps<typeof BaseSelect.Trigger> & {
  size?: ControlSize;
};

function SelectTrigger({
  className,
  children,
  ref,
  size = "md",
  style,
  ...props
}: SelectTriggerProps) {
  const stylexProps = stylex.props(controlSizeStyle(size));
  return (
    <BaseSelect.Trigger
      {...props}
      ref={ref}
      className={cn(selectTriggerVariants(), stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
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
type SelectContentProps = ComponentProps<typeof BaseSelect.Popup> & {
  positionerProps?: Omit<
    ComponentProps<typeof BaseSelect.Positioner>,
    "children"
  >;
};

function SelectContent({
  className,
  children,
  positionerProps,
  ref,
  ...props
}: SelectContentProps) {
  const {
    align = "start",
    alignItemWithTrigger = false,
    className: positionerClassName,
    sideOffset = 4,
    ...restPositionerProps
  } = positionerProps ?? {};

  return (
    <BaseSelect.Portal container={getLayerRoot("overlay")}>
      <BaseSelect.Positioner
        align={align}
        alignItemWithTrigger={alignItemWithTrigger}
        sideOffset={sideOffset}
        className={cn(LAYER_SURFACE_CLASS, positionerClassName)}
        {...restPositionerProps}
      >
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
  const stylexProps = stylex.props(menuRowStyle("sm"));
  return (
    <BaseSelect.Item
      {...props}
      ref={ref}
      className={cn(stylexProps.className, menuRowStateClassName, className)}
      style={{ ...stylexProps.style, ...props.style }}
    >
      <span className="flex size-4 shrink-0 items-center justify-center">
        <BaseSelect.ItemIndicator>
          <Check className="size-3.5" />
        </BaseSelect.ItemIndicator>
      </span>
      <BaseSelect.ItemText>{children}</BaseSelect.ItemText>
    </BaseSelect.Item>
  );
}

export { Select, SelectContent, SelectItem, SelectTrigger, SelectValue };
