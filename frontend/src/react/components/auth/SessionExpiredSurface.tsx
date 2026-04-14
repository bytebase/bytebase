import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
import { useAuthStore } from "@/store";
import { SigninBridge } from "./SigninBridge";

export function SessionExpiredSurface({
  currentPath,
}: {
  currentPath: string;
}) {
  const { t } = useTranslation();

  return createPortal(
    <div
      data-session-expired-surface
      className="fixed inset-0"
      aria-modal="true"
      role="dialog"
    >
      <div className={`fixed inset-0 ${LAYER_BACKDROP_CLASS} bg-overlay/60`} />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <div
          className={`w-full max-w-xl rounded-sm bg-background p-6 shadow-lg ${LAYER_SURFACE_CLASS}`}
        >
          <SigninBridge currentPath={currentPath} />
          <div className="mt-4 flex justify-end gap-x-2">
            <Button variant="ghost" onClick={() => useAuthStore().logout()}>
              {t("common.logout")}
            </Button>
          </div>
        </div>
      </div>
    </div>,
    getLayerRoot("critical")
  );
}
