import { defineStore } from "pinia";
import { VueShellBridgeEvent } from "@/react/shell-bridge";
import type { NotificationCreate } from "@/types";

/**
 * Notification store — kept as a Pinia store for backward compatibility
 * with the 17 Vue-side pushNotification() callers. No internal queue: each
 * push dispatches a window CustomEvent that the React toast manager
 * (frontend/src/react/lib/toast.ts) catches and renders. The Pinia layer
 * dies entirely in Phase B3 (Pinia -> Zustand).
 */
export const useNotificationStore = defineStore("notification", {
  state: () => ({}),
  actions: {
    pushNotification(notification: NotificationCreate) {
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent<NotificationCreate>(
            VueShellBridgeEvent.notification,
            { detail: notification }
          )
        );
      }
    },
  },
});

export const pushNotification = (notification: NotificationCreate) => {
  useNotificationStore().pushNotification(notification);
};
