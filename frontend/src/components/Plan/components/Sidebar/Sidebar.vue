<template>
  <div v-if="shouldShow" class="flex flex-col gap-y-3 py-2 px-3">
    <p class="textinfolabel -mb-2">{{ $t("common.options") }}</p>
    <GhostSection v-if="shouldShowGhostSection" />
    <PreBackupSection v-if="shouldShowPreBackupSection" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePlanContext } from "../../logic";
import GhostSection from "./GhostSection";
import { provideGhostSettingContext } from "./GhostSection/context";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";

const { isCreating, plan, selectedSpec, events } = usePlanContext();

const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext({
    project: computed(() => plan.value.projectEntity),
    plan: plan,
    selectedSpec: selectedSpec,
    isCreating,
  });

const { shouldShow: shouldShowPreBackupSection, events: preBackupEvents } =
  providePreBackupSettingContext({
    project: computed(() => plan.value.projectEntity),
    plan: plan,
    selectedSpec: selectedSpec,
    isCreating,
  });

const shouldShow = computed(() => {
  return shouldShowGhostSection.value || shouldShowPreBackupSection.value;
});

preBackupEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});

ghostEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});
</script>
