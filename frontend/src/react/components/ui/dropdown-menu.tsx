import { Menu as BaseMenu } from "@base-ui/react/menu";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

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

// ---- Portal + Positioner + Popup ----
// The `z-50` on the Positioner matches the z-index used by Dialog/Select/Tooltip
// (see ui/dialog.tsx, ui/select.tsx, ui/tooltip.tsx). It must not be removed —
// Dialog's backdrop/popup are hardcoded at z-50, so a DropdownMenu without an
// explicit z-index renders *behind* any open Dialog regardless of DOM portal
// order (z-auto loses to z-50). Within the same z-layer, stacking falls back to
// DOM order, which correctly puts later-mounted portals on top.
// See BYT-9226 and PR #19824 for the original regression.
function DropdownMenuContent({
  className,
  children,
  sideOffset = 4,
  align = "end",
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Popup> & {
  sideOffset?: ComponentProps<typeof BaseMenu.Positioner>["sideOffset"];
  align?: ComponentProps<typeof BaseMenu.Positioner>["align"];
}) {
  return (
    <BaseMenu.Portal>
      <BaseMenu.Positioner
        sideOffset={sideOffset}
        align={align}
        className="z-50"
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
  return (
    <BaseMenu.Item
      ref={ref}
      className={cn(
        "relative flex items-center gap-x-2 px-3 py-2 text-sm cursor-pointer select-none",
        "hover:bg-control-bg focus:bg-control-bg outline-hidden",
        "data-highlighted:bg-control-bg",
        "data-disabled:pointer-events-none data-disabled:opacity-50",
        className
      )}
      {...props}
    >
      {children}
    </BaseMenu.Item>
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

export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
};
