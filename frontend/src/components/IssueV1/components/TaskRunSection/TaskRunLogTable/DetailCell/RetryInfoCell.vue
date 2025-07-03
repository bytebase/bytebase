<template>
  <span v-if="!isNaN(retryCount) && !isNaN(maxRetries)">
    {{
      $t("issue.task-run.task-run-log.retry-info", {
        n: retryCount,
        m: maxRetries,
      })
    }}
  </span>
  <span v-else class="text-control-placeholder">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

const props = defineProps<{
  entry: FlattenLogEntry;
}>();

const { retryCount, maxRetries } = computed(() => {
  const { type, retryInfo } = props.entry;
  if (type === TaskRunLogEntry_Type.RETRY_INFO && retryInfo) {
    return {
      retryCount: retryInfo.retryCount,
      maxRetries: retryInfo.maximumRetries,
    };
  }
  return { retryCount: Number.NaN, maxRetries: Number.NaN };
}).value;
</script>
