import {
  Notification,
  NewNotification,
  NotificationFilter,
  NotificationState,
} from "../../types";
import { v1 as uuidv1 } from "uuid";

const state: () => NotificationState = () => ({
  notificationByModule: new Map(),
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

const getters = {};

const actions = {
  pushNotification({ commit }: any, newNotification: NewNotification) {
    const notification: Notification = {
      id: uuidv1(),
      createdTs: Date.now(),
      ...newNotification,
    };
    commit("appendNotification", notification);
  },

  tryPopNotification({ state, commit }: any, filter: NotificationFilter) {
    const notification = findNotification(state, filter);
    if (notification) {
      commit("removeNotification", notification);
    }
    return notification;
  },
};

const mutations = {
  appendNotification(state: NotificationState, notification: Notification) {
    const list = state.notificationByModule.get(notification.module);
    if (list) {
      list.push(notification);
    } else {
      state.notificationByModule.set(notification.module, [notification]);
    }
  },

  removeNotification(state: NotificationState, notification: Notification) {
    const list = state.notificationByModule.get(notification.module);
    if (list) {
      const i = list.indexOf(notification);
      if (i > -1) {
        list.splice(i, 1);
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
