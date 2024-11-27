<template>
  <NButton
    v-if="advise"
    size="small"
    type="primary"
    @click="toggleGhost(advise.on)"
  >
    {{ advise.text() }}
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { type PlanCheckDetailTableRow } from "@/components/PlanCheckRun/PlanCheckRunDetail.vue";
import { isGroupingChangeTaskV1, useIssueContext } from "../../logic";
import { allowGhostForTask } from "../Sidebar/GhostSection/common";

const emit = defineEmits<{
  (event: "toggle", on: boolean): void;
}>();

const props = defineProps<{
  row: PlanCheckDetailTableRow;
}>();

const { t } = useI18n();
const { isCreating, issue, selectedTask } = useIssueContext();

const code = computed(() => {
  return props.row.checkResult.sqlReviewReport?.code;
});

const advise = computed(() => {
  if (!isCreating.value) {
    return undefined;
  }
  if (!code.value) {
    return undefined;
  }
  if (isGroupingChangeTaskV1(issue.value, selectedTask.value)) {
    return undefined;
  }
  if (!allowGhostForTask(issue.value, selectedTask.value)) {
    return undefined;
  }
  if (code.value === 1801) {
    return {
      text: () => t("task.online-migration.enable"),
      on: true,
    };
  }
  if (code.value === 1803) {
    return {
      text: () => t("task.online-migration.disable"),
      on: false,
    };
  }
  return undefined;
});

const toggleGhost = (on: boolean) => {
  emit("toggle", on);
};
</script>
