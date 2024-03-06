<template>
  <div v-if="allowEnableGhost">
    <div class="normal-link inline-block" @click="enableGhost">
      {{ $t("task.online-migration.enable") }}
    </div>
  </div>
</template>

<script setup lang="ts">
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
