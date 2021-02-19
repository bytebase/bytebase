<template>
  <BannerDemo v-if="isDemo()" />
  <div class="relative h-screen overflow-hidden flex flex-col">
    <nav class="bg-white border-b border-block-border">
      <div class="max-w-full mx-auto px-6">
        <DashboardHeader />
      </div>
    </nav>
    <router-view name="body" />
  </div>
  <BBNotification
    :showing="state.notification != null"
    :style="state.notification?.style || 'INFO'"
    :title="state.notification?.title || ''"
    :description="state.notification?.description || ''"
  />
</template>

<script lang="ts">
import { reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import DashboardHeader from "../views/DashboardHeader.vue";
import BannerDemo from "../views/BannerDemo.vue";
import { Notification } from "../types";
import { isDemo } from "../utils";

const NOTIFICAITON_DURATION = 4000;

interface LocalState {
  notification?: Notification | null;
}

export default {
  name: "DashboardLayout",
  components: { DashboardHeader, BannerDemo },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      notification: null,
    });

    const watchNotification = () => {
      store
        .dispatch("notification/peekNotification", {
          module: "bytebase",
        })
        .then((notification) => {
          if (notification) {
            state.notification = notification;
            setTimeout(() => {
              store
                .dispatch("notification/popNotification", {
                  module: "bytebase",
                })
                .then(() => {
                  state.notification = null;
                });
            }, NOTIFICAITON_DURATION);
          }
        });
    };

    watchEffect(watchNotification);

    return {
      state,
      isDemo,
    };
  },
};
</script>
