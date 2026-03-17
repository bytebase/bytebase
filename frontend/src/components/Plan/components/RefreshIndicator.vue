<template>
  <div class="hidden sm:flex items-center gap-2">
    <NTooltip v-if="lastRefreshDisplay && lastRefreshAbsolute">
      <template #trigger>
        <span class="text-xs text-gray-400">
          {{
            $t("plan.refresh-indicator.last-refresh", { time: lastRefreshDisplay })
          }}
        </span>
      </template>
      <span>{{ lastRefreshAbsolute }}</span>
    </NTooltip>
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
import { RefreshCcwIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, onUnmounted, ref } from "vue";
import { formatAbsoluteDateTime, formatRelativeTime } from "@/utils";
import { useResourcePoller } from "../logic/poller";

const currentTime = ref(Date.now());
const REFRESH_INDICATOR_NOW_THRESHOLD_MS = 3_000;

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
const lastRefreshAbsolute = computed(() => {
  if (!lastRefreshTime.value) return null;
  return formatAbsoluteDateTime(lastRefreshTime.value);
});

const lastRefreshDisplay = computed(() => {
  if (!lastRefreshTime.value) return null;

  // Force reactivity by using currentTime (triggers every second)
  void currentTime.value;

  const diff = Date.now() - lastRefreshTime.value;
  if (diff < 3_600_000) {
    return formatRelativeTime(lastRefreshTime.value, {
      nowThresholdMs: REFRESH_INDICATOR_NOW_THRESHOLD_MS,
    });
  }
  return formatAbsoluteDateTime(lastRefreshTime.value);
});

const handleRefresh = async () => {
  await resourcePoller.refreshResources(
    ["issue", "plan", "planCheckRuns", "rollout", "taskRuns"],
    true /** force */
  );
};
</script>
