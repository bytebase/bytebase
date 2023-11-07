<template>
  <NSwitch
    :value="checked"
    :disabled="!allowChange"
    :loading="isUpdating"
    class="bb-ghost-switch"
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
import { hasFeature } from "@/store";
import { useIssueGhostContext } from "./common";

const { isCreating, issue, selectedTask: task } = useIssueContext();
const { viewType, toggleGhost, showFeatureModal } = useIssueGhostContext();
const isUpdating = ref(false);

const allowChange = computed(() => {
  return isCreating.value;
});

const checked = computed(() => {
  return viewType.value === "ON";
});

const toggleChecked = async (on: boolean) => {
  if (!hasFeature("bb.feature.online-migration")) {
    showFeatureModal.value = true;
    return;
  }

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

<style lang="postcss" scoped>
.bb-ghost-switch {
  --n-width: max(
    var(--n-rail-width),
    calc(var(--n-rail-width) + var(--n-button-width) - var(--n-rail-height))
  ) !important;
}
.bb-ghost-switch :deep(.n-switch__checked) {
  padding-right: calc(var(--n-rail-height) - var(--n-offset) + 1px);
}
.bb-ghost-switch :deep(.n-switch__unchecked) {
  padding-left: calc(var(--n-rail-height) - var(--n-offset) + 1px);
}
.bb-ghost-switch :deep(.n-switch__button-placeholder) {
  width: calc(1.25 * var(--n-rail-height));
}
</style>
