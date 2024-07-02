<template>
  <NButton
    v-if="allowEnableGhost"
    size="small"
    type="primary"
    @click="enableGhost"
  >
    {{ $t("task.online-migration.enable") }}
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import { isGroupingChangeTaskV1, useIssueContext } from "../../logic";
import { type PlanCheckDetailTableRow } from "../PlanCheckSection/PlanCheckBar/PlanCheckDetail.vue";
import { allowGhostForTask } from "../Sidebar/GhostSection/common";

const emit = defineEmits<{
  (event: "toggle", on: boolean): void;
}>();

defineProps<{
  row: PlanCheckDetailTableRow;
}>();

const { isCreating, issue, activeTask } = useIssueContext();

const allowEnableGhost = computed(() => {
  if (!isCreating.value) {
    return false;
  }
  if (isGroupingChangeTaskV1(issue.value, activeTask.value)) {
    return false;
  }

  if (!allowGhostForTask(issue.value, activeTask.value)) {
    return false;
  }

  return true;
});

const enableGhost = () => {
  emit("toggle", true);
};
</script>
