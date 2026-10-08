import { Settings } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/app/router/handles";
import { Button } from "@/components/ui/button";
import { FeatureModal } from "@/components/ui/feature-modal";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { ConnectionPane } from "./ConnectionPane/ConnectionPane";

/**
 * Right-side Sheet hosting the `ConnectionPane`.
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

  // Escape doesn't close the drawer, but a mask click does: cancel Base UI's
  // close handling when the reason is `escape-key`; outside-press (mask
  // click) is allowed through.
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
                    appearance="secondary"
                    size="sm"
                    className={cn("w-7 p-1")}
                    aria-label={t("sql-editor.manage-connections")}
                    // Just navigate. The route change unmounts the SQL
                    // editor; pre-closing the drawer adds an unnecessary
                    // close transition.
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
