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
          className={`fixed inset-0 ${LAYER_SURFACE_CLASS} flex items-center justify-center outline-none`}
        >
          <BaseDialog.Title className="sr-only">
            {t("auth.token-expired-title")}
          </BaseDialog.Title>
          <SigninBridge currentPath={currentPath} />
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  );
}
