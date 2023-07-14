<template>
  <div v-if="show" class="issue-debug">
    <heroicons:ellipsis-vertical-solid class="w-4 h-4" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { Task } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../../logic";

defineProps<{
  task: Task;
}>();

const { isCreating } = useIssueContext();

const show = computed(() => {
  if (isCreating.value) {
    return false;
  }

  // TODO
  return true;
});
</script>
