import { pushReactNotification } from "@/react/lib/toast";
import type { AppSliceCreator, NotificationSlice } from "./types";

export const createNotificationSlice: AppSliceCreator<
  NotificationSlice
> = () => ({
  notify: (notification) => {
    pushReactNotification(notification);
  },
});
