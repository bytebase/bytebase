<template>
  <div v-if="show" class="issue-debug">
    <heroicons:ellipsis-vertical-solid class="w-4 h-4" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { Task } from "@/types/proto/v1/rollout_service";
import {
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "../../logic";

const props = defineProps<{
  task: Task;
}>();

const { isCreating, activeTask, issue } = useIssueContext();

const actionList = computed(() => {
  return getApplicableTaskRolloutActionList(
    issue.value,
    props.task,
    true /* allowSkipPendingTask */
  );
});

const show = computed(() => {
  if (isCreating.value) {
    return false;
  }
  if (props.task.uid !== activeTask.value.uid) {
    return false;
  }

  return actionList.value.length > 0;
});
</script>
