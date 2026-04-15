import { Dialog as BaseDialog } from "@base-ui/react/dialog";
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

  return (
    <BaseDialog.Root open onOpenChange={() => {}}>
      <BaseDialog.Portal container={getLayerRoot("critical")}>
        <BaseDialog.Backdrop
          className={`fixed inset-0 ${LAYER_BACKDROP_CLASS} bg-overlay/60`}
        />
        <BaseDialog.Popup
          data-session-expired-surface
          className={`fixed left-1/2 top-1/2 ${LAYER_SURFACE_CLASS} w-full max-w-xl -translate-x-1/2 -translate-y-1/2 rounded-sm bg-background p-6 shadow-lg`}
        >
          <BaseDialog.Title className="sr-only">
            {t("auth.token-expired-title")}
          </BaseDialog.Title>
          <SigninBridge currentPath={currentPath} />
          <div className="mt-4 flex justify-end gap-x-2">
            <Button variant="ghost" onClick={() => useAuthStore().logout()}>
              {t("common.logout")}
            </Button>
          </div>
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  );
}
