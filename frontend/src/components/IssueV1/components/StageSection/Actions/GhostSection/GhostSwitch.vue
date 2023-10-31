<template>
  <NSwitch
    :value="checked"
    :disabled="!allowChange"
    :loading="isUpdating"
    @update:value="toggleChecked"
  >
    <template #checked>
      <span style="font-size: 10px">{{ $t("common.on") }}</span>
    </template>
    <template #unchecked>
      <span style="font-size: 10px">{{ $t("common.off") }}</span>
    </template>
  </NSwitch>
</template>

<script setup lang="ts">
import { NSwitch } from "naive-ui";
import { computed, ref } from "vue";
import { specForTask, useIssueContext } from "@/components/IssueV1/logic";
import { provideIssueGhostContext } from "./common";

const { isCreating, issue, selectedTask: task } = useIssueContext();
const { viewType, toggleGhost } = provideIssueGhostContext();
const isUpdating = ref(false);

const allowChange = computed(() => {
  return isCreating.value;
});

const checked = computed(() => {
  return viewType.value === "ON";
});

const toggleChecked = async (on: boolean) => {
  const spec = specForTask(issue.value.planEntity, task.value);
  if (!spec) return;
  isUpdating.value = true;
  try {
    await toggleGhost(spec, on);
  } finally {
    isUpdating.value = false;
  }
};
</script>
