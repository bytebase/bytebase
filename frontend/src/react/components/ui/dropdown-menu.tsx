import { Menu as BaseMenu } from "@base-ui/react/menu";
import * as stylex from "@stylexjs/stylex";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";
import { menuRowStateClassName, menuRowStyle } from "./styles.stylex";

// ---- Root ----
// Default to non-modal: row action menus should let users click through to
// other elements and dismiss by clicking outside, without locking page scroll.
// Callers that need modal behavior (rare) can pass `modal` explicitly.
function DropdownMenu({
  modal = false,
  ...props
}: ComponentProps<typeof BaseMenu.Root>) {
  return <BaseMenu.Root modal={modal} {...props} />;
}

// ---- Trigger ----
// Re-exported as-is so callers pass their own className/children. Base UI
// renders it as a <button> element by default.
const DropdownMenuTrigger = BaseMenu.Trigger;
const DropdownMenuSubmenu = BaseMenu.SubmenuRoot;

// ---- Portal + Positioner + Popup ----
function DropdownMenuContent({
  className,
  children,
  sideOffset = 4,
  align = "end",
  anchor,
  positionMethod,
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Popup> & {
  sideOffset?: ComponentProps<typeof BaseMenu.Positioner>["sideOffset"];
  align?: ComponentProps<typeof BaseMenu.Positioner>["align"];
  anchor?: ComponentProps<typeof BaseMenu.Positioner>["anchor"];
  positionMethod?: ComponentProps<typeof BaseMenu.Positioner>["positionMethod"];
}) {
  return (
    <BaseMenu.Portal container={getLayerRoot("overlay")}>
      <BaseMenu.Positioner
        sideOffset={sideOffset}
        align={align}
        anchor={anchor}
        positionMethod={positionMethod}
        className={LAYER_SURFACE_CLASS}
      >
        <BaseMenu.Popup
          ref={ref}
          className={cn(
            "min-w-[10rem] overflow-hidden rounded-sm border border-control-border bg-background py-1 shadow-md",
            "focus:outline-hidden",
            className
          )}
          {...props}
        >
          {children}
        </BaseMenu.Popup>
      </BaseMenu.Positioner>
    </BaseMenu.Portal>
  );
}

function DropdownMenuSubmenuContent({
  className,
  children,
  sideOffset = 4,
  align = "start",
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Popup> & {
  sideOffset?: ComponentProps<typeof BaseMenu.Positioner>["sideOffset"];
  align?: ComponentProps<typeof BaseMenu.Positioner>["align"];
}) {
  return (
    <BaseMenu.Portal container={getLayerRoot("overlay")}>
      <BaseMenu.Positioner
        side="right"
        sideOffset={sideOffset}
        align={align}
        className={LAYER_SURFACE_CLASS}
      >
        <BaseMenu.Popup
          ref={ref}
          className={cn(
            "min-w-[10rem] overflow-hidden rounded-sm border border-control-border bg-background py-1 shadow-md",
            "focus:outline-hidden",
            className
          )}
          {...props}
        >
          {children}
        </BaseMenu.Popup>
      </BaseMenu.Positioner>
    </BaseMenu.Portal>
  );
}

// ---- Item ----
function DropdownMenuItem({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Item>) {
  const stylexProps = stylex.props(menuRowStyle("sm"));
  return (
    <BaseMenu.Item
      {...props}
      ref={ref}
      className={cn(stylexProps.className, menuRowStateClassName, className)}
      style={{ ...stylexProps.style, ...props.style }}
    >
      {children}
    </BaseMenu.Item>
  );
}

function DropdownMenuSubmenuTrigger({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.SubmenuTrigger>) {
  const stylexProps = stylex.props(menuRowStyle("sm"));
  return (
    <BaseMenu.SubmenuTrigger
      {...props}
      ref={ref}
      className={cn(stylexProps.className, menuRowStateClassName, className)}
      style={{ ...stylexProps.style, ...props.style }}
    >
      {children}
    </BaseMenu.SubmenuTrigger>
  );
}

// ---- Separator ----
function DropdownMenuSeparator({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Separator>) {
  return (
    <BaseMenu.Separator
      ref={ref}
      className={cn("-mx-1 my-1 h-px bg-control-border", className)}
      {...props}
    />
  );
}

// ---- Label ----
// Non-interactive section header. Rendered as a plain <div> so grouped
// dropdown menus share the same typography without becoming focusable items.
function DropdownMenuLabel({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "px-2 py-1.5 text-xs font-semibold text-control-light",
        className
      )}
      {...props}
    />
  );
}

export {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSubmenu,
  DropdownMenuSubmenuContent,
  DropdownMenuSubmenuTrigger,
  DropdownMenuTrigger,
};
