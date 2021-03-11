<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <div>
        <ProvideContext>
          <div class="relative h-screen overflow-hidden flex flex-col">
            <BannerDemo v-if="isDemo()" />
            <nav class="bg-white border-b border-block-border">
              <div class="max-w-full mx-auto px-4">
                <DashboardHeader />
              </div>
            </nav>
            <router-view name="body" />
          </div>
        </ProvideContext>
      </div>
    </template>
    <template #fallback>
      <span>Loading...</span>
    </template>
  </Suspense>
  <BBNotification
    :placement="'BOTTOM_RIGHT'"
    :showing="state.notification != null"
    :style="state.notification?.style || 'INFO'"
    :title="state.notification?.title || ''"
    :description="state.notification?.description || ''"
    :payload="state.notification?.id"
    @close="removeNotification"
  />
</template>

<script lang="ts">
import { reactive, watchEffect, ref } from "vue";
import { useStore } from "vuex";
import ProvideContext from "../components/ProvideContext.vue";
import DashboardHeader from "../views/DashboardHeader.vue";
import BannerDemo from "../views/BannerDemo.vue";
import { Notification } from "../types";
import { isDemo } from "../utils";

const NOTIFICAITON_DURATION = 8000;

interface LocalState {
  notification?: Notification | undefined;
}

export default {
  name: "DashboardLayout",
  components: { ProvideContext, DashboardHeader, BannerDemo },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      notification: undefined,
    });

    const removeNotification = (id: string) => {
      store
        .dispatch("notification/removeNotification", {
          module: "bytebase",
          id,
        })
        .then(() => {
          state.notification = undefined;
        });
    };

    const watchNotification = () => {
      store
        .dispatch("notification/peekNotification", {
          module: "bytebase",
        })
        .then((notification: Notification) => {
          state.notification = notification;
          // We don't want user to miss "CRITICAL" notification and
          // thus don't automatically dismiss it.
          if (
            notification &&
            notification.style != "CRITICAL" &&
            !notification.manualHide
          ) {
            setTimeout(() => {
              removeNotification(notification.id);
            }, NOTIFICAITON_DURATION);
          }
        });
    };

    watchEffect(watchNotification);

    return {
      state,
      isDemo,
      removeNotification,
    };
  },
};
</script>
