<template>
  <div v-if="shouldShow" class="flex flex-col gap-y-3 py-2 px-3">
    <NTooltip :showArrow="false">
      <template #trigger>
        <p class="textinfolabel -mb-1">
          {{ $t("plan.options.self") }}
          <span class="opacity-80">
            ({{
              targetsForSpec(selectedSpec).length === flattenSpecCount
                ? $t("plan.options.applies-to-all-tasks")
                : $t("plan.options.applies-to-some-tasks", {
                    count: targetsForSpec(selectedSpec).length,
                    total: flattenSpecCount,
                  })
            }})
          </span>
        </p>
      </template>
      {{ $t("plan.options.split-into-multiple-issues-tip") }}
    </NTooltip>
    <GhostSection v-if="shouldShowGhostSection" />
    <PreBackupSection v-if="shouldShowPreBackupSection" />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { targetsForSpec, usePlanContext } from "../../logic";
import GhostSection from "./GhostSection";
import { provideGhostSettingContext } from "./GhostSection/context";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";

const { project } = useCurrentProjectV1();
const { isCreating, plan, selectedSpec, events } = usePlanContext();

const flattenSpecCount = computed(() =>
  plan.value.specs.reduce((acc, spec) => acc + targetsForSpec(spec).length, 0)
);

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
