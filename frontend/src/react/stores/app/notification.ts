import { emitReactNotification } from "@/react/shell-bridge";
import type { AppSliceCreator, NotificationSlice } from "./types";

export const createNotificationSlice: AppSliceCreator<NotificationSlice> = (
  set
) => ({
  notifications: [],

  notify: (notification) => {
    set((state) => ({
      notifications: [...state.notifications, notification],
    }));
    emitReactNotification(notification);
  },
});
