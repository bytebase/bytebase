<template>
  <BBModal
    :show="shouldShow"
    :trap-focus="true"
    :show-close="false"
    :mask-closable="false"
    :header-class="'hidden!'"
  >
    <div class="flex items-center w-auto md:min-w-96 max-w-full h-auto md:py-4">
      <div class="flex flex-col justify-center items-start flex-1 gap-y-4">
        <div>
          <p class="font-medium text-lg">
            {{ $t("auth.inactive-modal.title") }}
          </p>
          <span class="textinfo">
            {{
              $t("auth.inactive-modal.description", {
                minutes: showModalThresholdInMins,
              })
            }}
          </span>
        </div>
        <div class="w-full flex items-center justify-end gap-x-2">
          <NButton quaternary @click="logout">
            {{ $t("common.logout") }}
          </NButton>
          <NButton type="primary" @click="staySignedIn">
            {{ $t("auth.inactive-modal.stay-signed-in") }}
          </NButton>
        </div>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { computed, onUnmounted, ref, watch } from "vue";
import { BBModal } from "@/bbkit";
import { useCurrentTimestamp } from "@/composables/useCurrentTimestamp";
import { useLastActivity } from "@/composables/useLastActivity";
import { useAuthStore, useSettingV1Store } from "@/store";

const authStore = useAuthStore();
const settingStore = useSettingV1Store();
const timer = ref<NodeJS.Timeout | undefined>();

// Show the modal 3min before the inactiveTimeout threshold.
const showModalThresholdInMins = 3;

const logout = () => {
  authStore.logout();
};

const inactiveTimeoutInSeconds = computed(() =>
  Number(
    settingStore.workspaceProfileSetting?.inactiveSessionTimeout?.seconds ?? 0
  )
);

const { lastActivityTs } = useLastActivity();

const staySignedIn = () => (lastActivityTs.value = Date.now());

const { currentTsInMS } = useCurrentTimestamp();

const shouldShow = computed(() => {
  if (inactiveTimeoutInSeconds.value <= 0) {
    return false;
  }

  const inactiveInSeconds = (currentTsInMS.value - lastActivityTs.value) / 1000;
  // Show the modal 3min before the inactiveTimeout threshold.
  return (
    inactiveInSeconds >
    inactiveTimeoutInSeconds.value - showModalThresholdInMins * 60
  );
});

const resetTimeout = () => {
  if (timer.value) {
    clearTimeout(timer.value);
    timer.value = undefined;
  }
};

watch(
  () => shouldShow.value,
  (show) => {
    resetTimeout();

    if (show) {
      timer.value = setTimeout(
        () => {
          logout();
        },
        showModalThresholdInMins * 60 * 1000
      );
    }
  }
);

onUnmounted(() => {
  resetTimeout();
});
</script>
