import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import type { ComponentProps } from "react";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";

export const AgentDialog = BaseDialog.Root;

export const AgentDialogTrigger = BaseDialog.Trigger;

function AgentDialogOverlay({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Backdrop>) {
  return (
    <BaseDialog.Backdrop
      ref={ref}
      className={cn(
        `fixed inset-0 ${LAYER_BACKDROP_CLASS} bg-overlay/50`,
        className
      )}
      {...props}
    />
  );
}

export function AgentDialogContent({
  className,
  children,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Popup>) {
  return (
    <BaseDialog.Portal container={getLayerRoot("agent")}>
      <AgentDialogOverlay />
      <BaseDialog.Popup
        ref={ref}
        data-agent-dialog-content
        className={cn(
          `fixed left-1/2 top-1/2 ${LAYER_SURFACE_CLASS} -translate-x-1/2 -translate-y-1/2`,
          "w-[calc(100vw-8rem)] max-w-3xl 2xl:max-w-[55vw]",
          "max-h-[calc(100vh-10rem)] overflow-y-auto",
          "rounded-sm bg-background shadow-lg",
          className
        )}
        {...props}
      >
        {children}
      </BaseDialog.Popup>
    </BaseDialog.Portal>
  );
}

export function AgentDialogTitle({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Title>) {
  return (
    <BaseDialog.Title
      ref={ref}
      className={cn("text-lg font-semibold", className)}
      {...props}
    />
  );
}

export function AgentDialogDescription({
  className,
  ref,
  ...props
}: ComponentProps<typeof BaseDialog.Description>) {
  return (
    <BaseDialog.Description
      ref={ref}
      className={cn("text-sm text-control-light", className)}
      {...props}
    />
  );
}

export const AgentDialogClose = BaseDialog.Close;
