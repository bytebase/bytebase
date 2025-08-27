<template>
  <div v-if="shouldShow" class="flex flex-col gap-2">
    <p class="text-base">
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
import { useSelectedSpec } from "../SpecDetailView/context";
import GhostSection from "./GhostSection";
import { provideGhostSettingContext } from "./GhostSection/context";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";

const { project } = useCurrentProjectV1();
const { isCreating, plan, events, issue, rollout, readonly } = usePlanContext();
const selectedSpec = useSelectedSpec();

const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext({
    project,
    plan,
    selectedSpec,
    isCreating,
    issue,
    rollout,
    readonly,
  });

const { shouldShow: shouldShowPreBackupSection, events: preBackupEvents } =
  providePreBackupSettingContext({
    project,
    plan,
    selectedSpec,
    isCreating,
    issue,
    rollout,
    readonly,
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
