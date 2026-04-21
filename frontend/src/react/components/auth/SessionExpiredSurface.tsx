import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { useTranslation } from "react-i18next";
import {
  getLayerRoot,
  LAYER_BACKDROP_CLASS,
  LAYER_SURFACE_CLASS,
} from "@/react/components/ui/layer";
import { SigninPage } from "@/react/pages/auth/SigninPage";
import { useAuthStore } from "@/store";

export function SessionExpiredSurface({
  currentPath,
}: {
  currentPath: string;
}) {
  const { t } = useTranslation();

  const logoutFooter = (
    <div className="mt-4 flex justify-center">
      <button
        type="button"
        className="text-sm text-control-light hover:underline"
        onClick={() => useAuthStore().logout()}
      >
        {t("common.logout")}
      </button>
    </div>
  );

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
          <div
            className="bg-white shadow-lg rounded-sm py-3 flex pointer-events-auto flex-col gap-3"
            style={{
              maxWidth: "calc(100vw - 80px)",
              maxHeight: "calc(100vh - 80px)",
            }}
          >
            <div className="px-4 max-h-screen overflow-auto w-full h-full">
              <div className="flex items-center w-auto md:min-w-96 max-w-full h-auto md:py-4">
                <div className="flex flex-col justify-center items-center flex-1 gap-y-2">
                  <SigninPage
                    redirect={false}
                    redirectUrl={currentPath}
                    allowSignup={false}
                    hideFooter
                    footerOverride={logoutFooter}
                  />
                </div>
              </div>
            </div>
          </div>
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  );
}
