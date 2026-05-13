import {
  Toast as BaseToast,
  type ToastManagerAddOptions,
} from "@base-ui/react/toast";
import { VueShellBridgeEvent } from "@/react/shell-bridge";
import type { BBNotificationStyle, NotificationCreate } from "@/types";

const NOTIFICATION_DURATION_MS = 6000;
const CRITICAL_NOTIFICATION_DURATION_MS = 10000;

const STYLE_TO_TYPE: Record<
  BBNotificationStyle,
  "success" | "info" | "warning" | "error"
> = {
  SUCCESS: "success",
  INFO: "info",
  WARN: "warning",
  CRITICAL: "error",
};

export const toastManager = BaseToast.createToastManager();

type ToastOptions = ToastManagerAddOptions<Record<string, unknown>>;

export function mapNotificationToToast(item: NotificationCreate): ToastOptions {
  const type = STYLE_TO_TYPE[item.style];
  const priority: "low" | "high" = item.style === "CRITICAL" ? "high" : "low";
  const timeout = item.manualHide
    ? 0
    : item.style === "CRITICAL"
      ? CRITICAL_NOTIFICATION_DURATION_MS
      : NOTIFICATION_DURATION_MS;

  const actionProps =
    item.link && item.linkTitle
      ? {
          "aria-label": item.linkTitle,
          onClick: () => {
            window.open(item.link, "_blank", "noopener,noreferrer");
          },
          children: item.linkTitle,
        }
      : undefined;

  return {
    title: item.title,
    description:
      typeof item.description === "string" ? item.description : undefined,
    type,
    priority,
    timeout,
    actionProps,
  };
}

// Filter mirrors the previous Vue NotificationContext: only the "bytebase"
// module renders; other modules (e.g. agent) own their own UI surface.
export function pushReactNotification(item: NotificationCreate): void {
  if (item.module !== "bytebase") return;
  toastManager.add(mapNotificationToToast(item));
}

// Bridge from the Vue side: pushNotification() in the Pinia store dispatches
// a window CustomEvent. Register at module-eval so events fired during app
// bootstrap (e.g. auth/error interceptors before <Toaster /> mounts) are
// caught and queued in toastManager; the Provider drains the queue on mount.
// main.ts imports this module synchronously to make that ordering reliable.
if (typeof window !== "undefined") {
  const handler = (event: Event) => {
    const detail = (event as CustomEvent<NotificationCreate>).detail;
    if (detail) pushReactNotification(detail);
  };
  window.addEventListener(VueShellBridgeEvent.notification, handler);
  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      window.removeEventListener(VueShellBridgeEvent.notification, handler);
    });
  }
}
