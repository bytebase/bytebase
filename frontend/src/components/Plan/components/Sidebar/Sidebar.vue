<template>
  <div v-if="shouldShow" class="flex flex-col gap-y-3 py-2 px-3">
    <p class="textinfolabel -mb-2">{{ $t("common.options") }}</p>
    <PreBackupSection v-if="shouldShowPreBackupSection" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePlanContext } from "../../logic";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";

const { isCreating, plan, selectedSpec, events } = usePlanContext();

const { shouldShow: shouldShowPreBackupSection, events: preBackupEvents } =
  providePreBackupSettingContext({
    project: computed(() => plan.value.projectEntity),
    plan: plan,
    selectedSpec: selectedSpec,
    selectedTask: computed(() => undefined),
    isCreating,
  });

const shouldShow = computed(() => {
  return shouldShowPreBackupSection.value;
});

preBackupEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});
</script>
