<template>
  <div v-if="shouldShow" class="flex flex-col gap-y-3 px-4">
    <p class="text-base font-medium">
      {{ $t("plan.options.self") }}
    </p>
    <div class="w-auto">
      <GhostSection v-if="shouldShowGhostSection" />
      <PreBackupSection v-if="shouldShowPreBackupSection" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { usePlanContext } from "../../logic";
import { usePlanSpecContext } from "../SpecDetailView/context";
import GhostSection from "./GhostSection";
import { provideGhostSettingContext } from "./GhostSection/context";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";

const { project } = useCurrentProjectV1();
const { isCreating, plan, events } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();

const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext({
    project,
    plan: plan,
    selectedSpec,
    isCreating,
  });

const { shouldShow: shouldShowPreBackupSection, events: preBackupEvents } =
  providePreBackupSettingContext({
    project,
    plan: plan,
    selectedSpec,
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
