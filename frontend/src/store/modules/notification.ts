import { VueShellBridgeEvent } from "@/react/shell-bridge";
import type { NotificationCreate } from "@/types";

/**
 * Dispatches a notification as a window CustomEvent that the React toast
 * manager (frontend/src/react/lib/toast.ts) catches and renders. Pinia-free:
 * the former `useNotificationStore` was an empty-state shim around this same
 * dispatch and has been removed.
 */
export const pushNotification = (notification: NotificationCreate) => {
  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent<NotificationCreate>(VueShellBridgeEvent.notification, {
        detail: notification,
      })
    );
  }
};
