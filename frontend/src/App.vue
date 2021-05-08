<template>
  <router-view />
  <template v-if="state.notificationList.length > 0">
    <BBNotification
      :placement="'BOTTOM_RIGHT'"
      :notificationList="state.notificationList"
      @close="removeNotification"
    />
  </template>
</template>

<script lang="ts">
import { reactive, watchEffect, onErrorCaptured } from "vue";
import { useStore } from "vuex";
import ProvideContext from "./components/ProvideContext.vue";
import { isDev } from "./utils";
import { Notification } from "./types";
import { BBNotificationItem } from "./bbkit/types";

const NOTIFICAITON_DURATION = 8000;

interface LocalState {
  notificationList: BBNotificationItem[];
}

export default {
  name: "App",
  components: { ProvideContext },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      notificationList: [],
    });

    const removeNotification = (id: string) => {
      state.notificationList.shift();
    };

    const watchNotification = () => {
      store
        .dispatch("notification/tryPopNotification", {
          module: "bytebase",
        })
        .then((notification: Notification | undefined) => {
          if (notification) {
            state.notificationList.push({
              style: notification.style,
              title: notification.title,
              description: notification.description || "",
              link: notification.link || "",
              linkTitle: notification.linkTitle || "",
            });
            // state.notification = notification;
            // We don't want user to miss "CRITICAL" notification and
            // thus don't automatically dismiss it.
            if (notification.style != "CRITICAL" && !notification.manualHide) {
              setTimeout(() => {
                removeNotification(notification.id);
              }, NOTIFICAITON_DURATION);
            }
          }
        });
    };

    watchEffect(watchNotification);

    // Restore previously logged in user if applicable.
    store.dispatch("auth/restoreUser");

    onErrorCaptured((e: any, _, info) => {
      // If e has response, then we assume it's an http error and has already been
      // handled by the axios global handler.
      if (!e.response) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: `Internal error occured`,
          description: isDev() ? e.stack : undefined,
        });
      }
      return true;
    });

    return {
      state,
      removeNotification,
    };
  },
};
</script>
