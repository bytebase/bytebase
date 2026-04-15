import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { useTranslation } from "react-i18next";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
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
          className={`fixed inset-0 ${LAYER_SURFACE_CLASS} flex items-center justify-center p-4 outline-none`}
        >
          <BaseDialog.Title className="sr-only">
            {t("auth.token-expired-title")}
          </BaseDialog.Title>
          <div className="pointer-events-auto flex w-auto max-w-full items-center md:min-w-96 md:py-4">
            <div className="flex flex-1 flex-col items-center justify-center gap-y-2">
              <SigninBridge currentPath={currentPath} />
            </div>
          </div>
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  );
}
