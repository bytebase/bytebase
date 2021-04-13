<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannerDemo v-if="isDemo" />
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-4">
        <DashboardHeader />
      </div>
    </nav>
    <router-view name="body" />
  </div>
  <template v-if="state.notificationList.length > 0">
    <BBNotification
      :placement="'BOTTOM_RIGHT'"
      :notificationList="state.notificationList"
      @close="removeNotification"
    />
  </template>
</template>

<script lang="ts">
import { reactive, watchEffect, ref } from "vue";
import { useStore } from "vuex";
import ProvideContext from "../components/ProvideContext.vue";
import DashboardHeader from "../views/DashboardHeader.vue";
import BannerDemo from "../views/BannerDemo.vue";
import { Notification } from "../types";
import { BBNotificationItem } from "../bbkit/types";

const NOTIFICAITON_DURATION = 8000;

interface LocalState {
  notificationList: BBNotificationItem[];
}

export default {
  name: "DashboardLayout",
  components: { ProvideContext, DashboardHeader, BannerDemo },
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

    return {
      state,
      removeNotification,
    };
  },
};
</script>
