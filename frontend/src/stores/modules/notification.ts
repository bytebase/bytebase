import { pushReactNotification } from "@/lib/toast";
import type { NotificationCreate } from "@/types";

export const pushNotification = (notification: NotificationCreate) => {
  pushReactNotification(notification);
};
