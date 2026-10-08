import { Select as BaseSelect } from "@base-ui/react/select";
import * as stylex from "@stylexjs/stylex";
import { cva } from "class-variance-authority";
import { Check, ChevronDown, X } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type MouseEvent,
  type ReactNode,
  useContext,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";
import {
  type ControlSize,
  controlMinHeightStyle,
  controlSizeStyle,
  menuRowStateClassName,
  menuRowStyle,
  overlaySurfaceClassName,
} from "./styles.stylex";

// ---- Root ----
const SelectContext = createContext<{
  multiple: boolean;
  disabled: boolean;
  getLabel: (value: unknown) => ReactNode;
  removeValue: (value: unknown, event: MouseEvent<HTMLButtonElement>) => void;
} | null>(null);

function Select<Value, Multiple extends boolean | undefined = false>(
  props: BaseSelect.Root.Props<Value, Multiple>
) {
  type ChangedValue = Parameters<NonNullable<typeof props.onValueChange>>[0];
  const [uncontrolledValue, setUncontrolledValue] = useState(
    (props.defaultValue ?? (props.multiple ? [] : null)) as ChangedValue
  );
  const value = props.value === undefined ? uncontrolledValue : props.value;
  const disabled = !!(props.disabled || props.readOnly);

  const handleValueChange = (
    nextValue: ChangedValue,
    details: BaseSelect.Root.ChangeEventDetails
  ) => {
    props.onValueChange?.(nextValue, details);
    if (!details.isCanceled && props.value === undefined) {
      setUncontrolledValue(nextValue);
    }
  };

  const getLabel = (itemValue: unknown): ReactNode => {
    if (props.itemToStringLabel) {
      return props.itemToStringLabel(itemValue as Value);
    }
    if (itemValue && typeof itemValue === "object" && "label" in itemValue) {
      return itemValue.label as ReactNode;
    }
    if (Array.isArray(props.items)) {
      const items = props.items.flatMap((item) =>
        "items" in item ? item.items : [item]
      );
      return (
        items.find((item) =>
          props.isItemEqualToValue
            ? props.isItemEqualToValue(item.value, itemValue as Value)
            : Object.is(item.value, itemValue)
        )?.label ?? String(itemValue)
      );
    }
    return (
      (props.items as Record<string, ReactNode> | undefined)?.[
        String(itemValue)
      ] ?? String(itemValue)
    );
  };

  return (
    <SelectContext.Provider
      value={{
        multiple: !!props.multiple,
        disabled,
        getLabel,
        removeValue: (itemValue, event) => {
          if (disabled || !Array.isArray(value)) return;
          const details: BaseSelect.Root.ChangeEventDetails = {
            reason: "none",
            event: event.nativeEvent,
            trigger: event.currentTarget,
            isCanceled: false,
            isPropagationAllowed: false,
            cancel() {
              details.isCanceled = true;
            },
            allowPropagation() {
              details.isPropagationAllowed = true;
            },
          };
          handleValueChange(
            value.filter((item) => !Object.is(item, itemValue)) as ChangedValue,
            details
          );
        },
      }}
    >
      <BaseSelect.Root
        {...props}
        value={value}
        onValueChange={handleValueChange}
      />
    </SelectContext.Provider>
  );
}

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
  const context = useContext(SelectContext);
  const multiple = !!context?.multiple;
  const stylexProps = stylex.props(
    multiple ? controlMinHeightStyle(size) : controlSizeStyle(size)
  );
  return (
    <SelectContext.Provider
      value={
        context
          ? { ...context, disabled: context.disabled || !!props.disabled }
          : null
      }
    >
      <BaseSelect.Trigger
        {...props}
        // Chip removal buttons cannot be nested inside a native button.
        {...(multiple ? { render: <div />, nativeButton: false } : {})}
        ref={ref}
        className={cn(
          selectTriggerVariants(),
          stylexProps.className,
          multiple &&
            "py-1 aria-disabled:pointer-events-none aria-disabled:opacity-50",
          className
        )}
        style={{ ...stylexProps.style, ...style }}
      >
        {children}
        <BaseSelect.Icon>
          <ChevronDown className="size-3.5 opacity-50 shrink-0" />
        </BaseSelect.Icon>
      </BaseSelect.Trigger>
    </SelectContext.Provider>
  );
}

// ---- Value ----
function SelectValue({
  children,
  className,
  ...props
}: ComponentProps<typeof BaseSelect.Value>) {
  const context = useContext(SelectContext);
  const { t } = useTranslation();
  if (!context?.multiple || children != null) {
    return (
      <BaseSelect.Value {...props} className={className}>
        {children}
      </BaseSelect.Value>
    );
  }
  return (
    <BaseSelect.Value
      {...props}
      className={cn("flex min-w-0 flex-1 flex-wrap gap-1", className)}
    >
      {(values: unknown[]) =>
        !Array.isArray(values) || values.length === 0
          ? props.placeholder
          : values.map((value, index) => (
              <span
                key={index}
                className="inline-flex max-w-full min-w-0 items-center gap-x-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
              >
                <span className="min-w-0 truncate">
                  {context.getLabel(value)}
                </span>
                {!context.disabled && (
                  <button
                    type="button"
                    aria-label={`${t("common.remove")} ${String(context.getLabel(value))}`}
                    title={t("common.remove")}
                    className="shrink-0 hover:text-error focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
                    onPointerDown={(event) => event.stopPropagation()}
                    onMouseDown={(event) => event.stopPropagation()}
                    onKeyDown={(event) => event.stopPropagation()}
                    onClick={(event) => {
                      event.stopPropagation();
                      event.currentTarget
                        .closest<HTMLElement>('[role="combobox"]')
                        ?.focus();
                      context.removeValue(value, event);
                    }}
                  >
                    <X className="size-3" />
                  </button>
                )}
              </span>
            ))
      }
    </BaseSelect.Value>
  );
}

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
            "min-w-(--anchor-width)",
            overlaySurfaceClassName,
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
      <BaseSelect.ItemText className="min-w-0 flex-1">
        {children}
      </BaseSelect.ItemText>
      <span className="flex size-4 shrink-0 items-center justify-center">
        <BaseSelect.ItemIndicator>
          <Check className="size-3.5" />
        </BaseSelect.ItemIndicator>
      </span>
    </BaseSelect.Item>
  );
}

export { Select, SelectContent, SelectItem, SelectTrigger, SelectValue };
