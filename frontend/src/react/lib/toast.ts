import {
  Toast as BaseToast,
  type ToastManagerAddOptions,
} from "@base-ui/react/toast";
import type { NotificationCreate } from "@/types/notification";

const NOTIFICATION_DURATION_MS = 6000;
const CRITICAL_NOTIFICATION_DURATION_MS = 10000;

const VUE_NOTIFICATION_EVENT = "bb.vue-notification";

/**
 * Module-level toast manager — created once, lives outside any React tree.
 * Callers anywhere (React components, Zustand slices, plain TS modules,
 * the Vue side via the window-event bridge below) can call .add() / .close().
 *
 * The <Toaster /> component subscribes via Base UI's useToastManager() hook
 * and renders each toast.
 */
export const toastManager = BaseToast.createToastManager();

type ToastOptions = ToastManagerAddOptions<Record<string, unknown>>;

/**
 * Convert the project's NotificationCreate shape into Base UI Toast options.
 * Pure function — exported for testing.
 */
export function mapNotificationToToast(item: NotificationCreate): ToastOptions {
  const type =
    item.style === "SUCCESS"
      ? "success"
      : item.style === "WARN"
        ? "warning"
        : item.style === "CRITICAL"
          ? "error"
          : "info";
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

/**
 * Push a notification through the React toast renderer. Safe to call from
 * any context (component, store, plain TS module). Filters by
 * module === "bytebase" to match the previous Vue NotificationContext.
 */
export function pushReactNotification(item: NotificationCreate): void {
  if (item.module !== "bytebase") return;
  toastManager.add(mapNotificationToToast(item));
}

// Module-eval-time listener: catch notifications originating on the Vue side
// (Pinia notificationStore.pushNotification) and forward them to the toast
// manager. Registered exactly once per module load; main.ts imports this
// module during app bootstrap, before any pushNotification fires.
if (typeof window !== "undefined") {
  window.addEventListener(VUE_NOTIFICATION_EVENT, (event: Event) => {
    const detail = (event as CustomEvent<NotificationCreate>).detail;
    if (detail) pushReactNotification(detail);
  });
}
