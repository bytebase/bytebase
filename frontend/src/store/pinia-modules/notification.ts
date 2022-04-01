import { defineStore } from "pinia";
import { v1 as uuidv1 } from "uuid";
import {
  Notification,
  NotificationCreate,
  NotificationFilter,
  NotificationState,
} from "../../types";

export const useNotificationStore = defineStore("notification", {
  state: (): NotificationState => ({
    notificationByModule: new Map(),
  }),
  actions: {
    appendNotification(notification: Notification) {
      const list = this.notificationByModule.get(notification.module);
      if (list) {
        list.push(notification);
      } else {
        this.notificationByModule.set(notification.module, [notification]);
      }
    },
    removeNotification(notification: Notification) {
      const list = this.notificationByModule.get(notification.module);
      if (list) {
        const i = list.indexOf(notification);
        if (i > -1) {
          list.splice(i, 1);
        }
      }
    },
    pushNotification(notificationCreate: NotificationCreate) {
      const notification: Notification = {
        id: uuidv1(),
        createdTs: Date.now() / 1000,
        ...notificationCreate,
      };
      this.appendNotification(notification);
    },
    tryPopNotification(filter: NotificationFilter) {
      const notification = findNotification(this.$state, filter);
      if (notification) {
        this.removeNotification(notification);
      }
      return notification;
    },
  },
});

function findNotification(
  state: NotificationState,
  filter: NotificationFilter
): Notification | undefined {
  const list = state.notificationByModule.get(filter.module);
  if (list && list.length > 0) {
    if (filter.id) {
      return list.find((item) => item.id == filter.id);
    }
    return list[0];
  }
  return undefined;
}
