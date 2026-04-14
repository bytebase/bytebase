import { Menu as BaseMenu } from "@base-ui/react/menu";
import type { ComponentProps } from "react";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";

export function AgentDropdownMenu({
  modal = false,
  ...props
}: ComponentProps<typeof BaseMenu.Root>) {
  return <BaseMenu.Root modal={modal} {...props} />;
}

export const AgentDropdownMenuTrigger = BaseMenu.Trigger;

export function AgentDropdownMenuContent({
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
    <BaseMenu.Portal container={getLayerRoot("agent")}>
      <BaseMenu.Positioner
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

export function AgentDropdownMenuItem({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseMenu.Item>) {
  return (
    <BaseMenu.Item
      ref={ref}
      className={cn(
        "relative flex cursor-pointer items-center gap-x-2 px-3 py-2 text-sm select-none",
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

export function AgentDropdownMenuSeparator({
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
