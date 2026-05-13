import { defineStore } from "pinia";
import { v1 as uuidv1 } from "uuid";
import type { Notification, NotificationCreate } from "@/types";

const VUE_NOTIFICATION_EVENT = "bb.vue-notification";

/**
 * Notification store — kept as a Pinia store for backward compatibility
 * with the 17 Vue-side pushNotification() callers. Internally it no longer
 * queues anything; instead each push dispatches a window CustomEvent that
 * the React toast manager (frontend/src/react/lib/toast.ts) catches and
 * renders. Vue retains no renderer.
 *
 * This store dies entirely in Phase B3 (Pinia -> Zustand).
 */
export const useNotificationStore = defineStore("notification", {
  state: () => ({}),
  actions: {
    pushNotification(notificationCreate: NotificationCreate) {
      const notification: Notification = {
        id: uuidv1(),
        createdTs: Date.now() / 1000,
        ...notificationCreate,
      };
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent<Notification>(VUE_NOTIFICATION_EVENT, {
            detail: notification,
          })
        );
      }
    },
  },
});

export const pushNotification = (notificationCreate: NotificationCreate) => {
  useNotificationStore().pushNotification(notificationCreate);
};
