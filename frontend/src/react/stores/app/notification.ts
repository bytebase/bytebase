import type { AppSliceCreator, NotificationSlice } from "./types";

export const createNotificationSlice: AppSliceCreator<NotificationSlice> = (
  set
) => ({
  notifications: [],

  notify: (notification) => {
    set((state) => ({
      notifications: [...state.notifications, notification],
    }));
    window.dispatchEvent(
      new CustomEvent("bb.react-notification", { detail: notification })
    );
  },
});
