import { ContextMenu as BaseContextMenu } from "@base-ui/react/context-menu";
import * as stylex from "@stylexjs/stylex";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import {
  getLayerRoot,
  LAYER_SURFACE_CLASS,
  usePreserveHigherLayerAccess,
} from "./layer";
import {
  menuRowStateClassName,
  menuRowStyle,
  overlaySurfaceClassName,
} from "./styles.stylex";

// ---- Root ----
// Wraps BaseContextMenu.Root. ContextMenu has no `modal` prop (it is omitted
// by Base UI); right-click menus are inherently non-modal.
const ContextMenu = BaseContextMenu.Root;

// ---- Trigger ----
// Renders a <div> by default. Wrap the target element to enable right-click.
const ContextMenuTrigger = BaseContextMenu.Trigger;

// ---- Portal + Positioner + Popup ----
function ContextMenuContent({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseContextMenu.Popup>) {
  usePreserveHigherLayerAccess("overlay");

  return (
    <BaseContextMenu.Portal container={getLayerRoot("overlay")}>
      <BaseContextMenu.Positioner className={LAYER_SURFACE_CLASS}>
        <BaseContextMenu.Popup
          ref={ref}
          className={cn("min-w-[10rem]", overlaySurfaceClassName, className)}
          {...props}
        >
          {children}
        </BaseContextMenu.Popup>
      </BaseContextMenu.Positioner>
    </BaseContextMenu.Portal>
  );
}

// ---- Item ----
function ContextMenuItem({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseContextMenu.Item>) {
  const stylexProps = stylex.props(menuRowStyle("sm"));
  return (
    <BaseContextMenu.Item
      {...props}
      ref={ref}
      className={cn(stylexProps.className, menuRowStateClassName, className)}
      style={{ ...stylexProps.style, ...props.style }}
    >
      {children}
    </BaseContextMenu.Item>
  );
}

// ---- Separator ----
function ContextMenuSeparator({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseContextMenu.Separator>) {
  return (
    <BaseContextMenu.Separator
      ref={ref}
      className={cn("my-1 border-t border-control-border", className)}
      {...props}
    />
  );
}

// ---- Label ----
// Non-interactive section header. Not a Base UI component — rendered as a
// plain <div> since Base UI's ContextMenu has no GroupLabel styled element.
function ContextMenuLabel({
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
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
};
