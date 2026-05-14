import { Settings } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { FeatureModal } from "@/react/components/ui/feature-modal";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { ConnectionPane } from "./ConnectionPane/ConnectionPane";

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPanel.vue.
 * Right-side Sheet hosting the `ConnectionPane`. The open/close state is
 * read directly from `useSQLEditorStore(s => s.showConnectionPanel)`
 * rather than passed through `<ReactPageMount>` props —
 * funneling boolean state through Vue→React attrs proved brittle (Vue's
 * template compiler doesn't auto-translate `(v) => (ref = v)` into
 * `.value = v` outside of `v-model`, and concurrent renders can leave the
 * Sheet's `open` prop stale after a programmatic close).
 *
 * The outer Vue drawer used a dynamic `width` (50vw up to 800px, falling
 * back to `calc(100vw - 4rem)` on narrow viewports). The shadcn Sheet's
 * `wide` tier (832px) is the closest match; it clamps via max-w-[100vw]
 * so narrow viewports remain safe.
 */
export function ConnectionPanel() {
  const { t } = useTranslation();
  const open = useSQLEditorStore((s) => s.showConnectionPanel);
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const allowManageInstance = hasWorkspacePermissionV2("bb.instances.list");
  // Hoisted from ConnectionPaneInner so the FeatureModal portal mounts as
  // a SIBLING of the drawer Sheet rather than a descendant. With both at
  // the same nesting level the FeatureModal's overlay reliably stacks
  // above the Sheet's overlay regardless of Base UI portal scheduling.
  const [missingFeature, setMissingFeature] = useState<PlanFeature | undefined>(
    undefined
  );

  // Vue had `:close-on-esc="false"` (mask click still closed the drawer).
  // Mirror that by canceling Base UI's close handling when the reason is
  // `escapeKey`; outside-press (mask click) is allowed through.
  const handleOpenChange = (
    next: boolean,
    eventDetails?: { reason?: string; cancel?: () => void }
  ) => {
    if (!next && eventDetails?.reason === "escape-key") {
      eventDetails.cancel?.();
      return;
    }
    setShowConnectionPanel(next);
  };

  return (
    <>
      <Sheet open={open} onOpenChange={handleOpenChange}>
        <SheetContent width="wide" className="p-0">
          <SheetHeader>
            <div className="flex items-center gap-x-1">
              <SheetTitle>{t("database.select")}</SheetTitle>
              {allowManageInstance && (
                <Tooltip content={t("sql-editor.manage-connections")}>
                  <Button
                    variant="ghost"
                    size="sm"
                    className={cn("size-7 p-1")}
                    aria-label={t("sql-editor.manage-connections")}
                    // Match Vue: just navigate. The route change unmounts
                    // the SQL editor; pre-closing the drawer added an
                    // unnecessary close transition.
                    onClick={() => {
                      void router.push({ name: INSTANCE_ROUTE_DASHBOARD });
                    }}
                  >
                    <Settings className="size-4" />
                  </Button>
                </Tooltip>
              )}
            </div>
          </SheetHeader>
          <SheetBody className="p-0">
            <ConnectionPane show={open} onMissingFeature={setMissingFeature} />
          </SheetBody>
        </SheetContent>
      </Sheet>
      {/* Sibling-of-Sheet placement: ensures the modal portal mounts after
          the Sheet's portal in the overlay layer, so its backdrop reliably
          stacks ABOVE the drawer. */}
      <FeatureModal
        open={!!missingFeature}
        feature={missingFeature}
        onOpenChange={(o) => {
          if (!o) setMissingFeature(undefined);
        }}
      />
    </>
  );
}
