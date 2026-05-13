import { Toast as BaseToast } from "@base-ui/react/toast";
import { useTranslation } from "react-i18next";
import { toastManager } from "@/react/lib/toast";
import { getLayerRoot, LAYER_Z_INDEX } from "./layer";
import {
  ToastAction,
  ToastClose,
  ToastDescription,
  ToastRoot,
  ToastTitle,
  type ToastVariant,
} from "./toast";

const TOAST_LIMIT = 5;

// Map Base UI's type string (which we set in toast.ts) onto our visual variant.
function variantFromType(type: string | undefined): ToastVariant {
  if (type === "success" || type === "warning" || type === "error") {
    return type;
  }
  return "info";
}

function ToastList() {
  const { toasts } = BaseToast.useToastManager();
  const { t } = useTranslation();
  return (
    <>
      {toasts.map((toast) => (
        <ToastRoot
          key={toast.id}
          toast={toast}
          variant={variantFromType(toast.type)}
        >
          <ToastClose aria-label={t("common.close")} />
          {toast.title ? <ToastTitle>{toast.title}</ToastTitle> : null}
          {toast.description ? (
            <ToastDescription>{toast.description}</ToastDescription>
          ) : null}
          {toast.actionProps ? <ToastAction {...toast.actionProps} /> : null}
        </ToastRoot>
      ))}
    </>
  );
}

/**
 * The Toaster shell — mounted once, persistent for the app lifetime.
 *
 * Structure: Provider supplies the context bound to the standalone
 * toastManager. The whole Viewport is portaled into getLayerRoot("overlay")
 * so toasts inherit the overlay family's aria-hidden / inert behavior
 * (e.g. session-expired surface at the 'critical' layer obscures them).
 */
export function Toaster() {
  return (
    <BaseToast.Provider toastManager={toastManager} limit={TOAST_LIMIT}>
      <BaseToast.Portal container={getLayerRoot("overlay")}>
        <BaseToast.Viewport
          className="fixed bottom-4 right-4 flex w-(--toast-width) flex-col gap-2"
          style={{
            // Tailwind v4 reads CSS vars; expose toast width here so the
            // toast card class can reference it. Width matches naive-ui's
            // default.
            ["--toast-width" as string]: "24rem",
            zIndex: LAYER_Z_INDEX.overlay,
          }}
        >
          <ToastList />
        </BaseToast.Viewport>
      </BaseToast.Portal>
    </BaseToast.Provider>
  );
}
