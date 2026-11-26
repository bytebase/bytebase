import { PreviewCard as BasePreviewCard } from "@base-ui/react/preview-card";
import type { ComponentProps, ReactNode } from "react";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";

export function AgentPersistentPreview({
  anchor,
  children,
}: Pick<ComponentProps<typeof BasePreviewCard.Positioner>, "anchor"> & {
  children: ReactNode;
}) {
  // The PreviewCard portal leaves the persistent surface in the normal tab order.
  return (
    <BasePreviewCard.Root
      open
      onOpenChange={(_, details) => {
        details.cancel();
        details.allowPropagation();
      }}
    >
      <BasePreviewCard.Portal container={getLayerRoot("agent")}>
        <BasePreviewCard.Positioner
          anchor={anchor}
          side="top"
          align="end"
          sideOffset={4}
          positionMethod="fixed"
          className={LAYER_SURFACE_CLASS}
        >
          {children}
        </BasePreviewCard.Positioner>
      </BasePreviewCard.Portal>
    </BasePreviewCard.Root>
  );
}
