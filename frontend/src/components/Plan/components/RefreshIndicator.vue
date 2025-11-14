<template>
  <div class="hidden sm:flex items-center gap-2">
    <span v-if="lastRefreshDisplay" class="text-xs text-gray-400">
      {{
        $t("plan.refresh-indicator.last-refresh", { time: lastRefreshDisplay })
      }}
    </span>
    <NButton
      size="tiny"
      quaternary
      :loading="isRefreshing"
      :disabled="isRefreshing"
      @click="handleRefresh"
    >
      <template #icon>
        <RefreshCcwIcon class="w-4 h-4" />
      </template>
      <span>{{ $t("common.refresh") }}</span>
    </NButton>
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { RefreshCcwIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useResourcePoller } from "../logic/poller";

const { t } = useI18n();
const currentTime = ref(Date.now());

const resourcePoller = useResourcePoller();

const isRefreshing = computed(() => {
  return resourcePoller.isRefreshing.value;
});

// Update current time every second to keep the display fresh
const timer = ref(
  setInterval(() => {
    currentTime.value = Date.now();
  }, 1000)
);

// Clean up timer on unmount
onUnmounted(() => {
  clearInterval(timer.value);
});

const lastRefreshTime = computed(() => resourcePoller.lastRefreshTime.value);

const lastRefreshDisplay = computed(() => {
  if (!lastRefreshTime.value) return null;

  // Force reactivity by using currentTime (triggers every second)
  void currentTime.value;

  const now = Date.now();
  const diff = now - lastRefreshTime.value;

  if (diff < 10000) {
    return t("common.just-now");
  } else if (diff < 60000) {
    const seconds = Math.floor(diff / 1000);
    return t("common.n-seconds-ago", { count: seconds });
  }
  return dayjs(lastRefreshTime.value).format("YYYY-MM-DD HH:mm:ss");
});

const handleRefresh = async () => {
  await resourcePoller.refreshResources(
    ["issue", "plan", "planCheckRuns", "rollout", "taskRuns"],
    true /** force */
  );
};
</script>
