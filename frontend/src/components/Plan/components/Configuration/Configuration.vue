<template>
  <div v-if="shouldShow" class="w-full">
    <div class="mb-4">
      <h3 class="text-lg font-medium text-main">
        {{ $t("plan.options.self") }}
      </h3>
    </div>
    <div class="space-y-4">
      <InstanceRoleSection v-if="shouldShowInstanceRoleSection" />
      <TransactionModeSection v-if="shouldShowTransactionModeSection" />
      <IsolationLevelSection v-if="shouldShowIsolationLevelSection" />
      <PreBackupSection v-if="shouldShowPreBackupSection" />
      <GhostSection v-if="shouldShowGhostSection" />
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
import InstanceRoleSection from "./InstanceRoleSection";
import { provideInstanceRoleSettingContext } from "./InstanceRoleSection/context";
import IsolationLevelSection from "./IsolationLevelSection";
import { provideIsolationLevelSettingContext } from "./IsolationLevelSection/context";
import PreBackupSection from "./PreBackupSection";
import { providePreBackupSettingContext } from "./PreBackupSection/context";
import TransactionModeSection from "./TransactionModeSection";
import { provideTransactionModeSettingContext } from "./TransactionModeSection/context";

const { project } = useCurrentProjectV1();
const { isCreating, plan, events, issue, rollout, readonly } = usePlanContext();
const selectedSpec = useSelectedSpec();

const {
  shouldShow: shouldShowTransactionModeSection,
  events: transactionModeEvents,
} = provideTransactionModeSettingContext({
  project,
  plan,
  selectedSpec,
  isCreating,
  issue,
  rollout,
  readonly,
});

const {
  shouldShow: shouldShowInstanceRoleSection,
  events: instanceRoleEvents,
} = provideInstanceRoleSettingContext({
  project,
  plan,
  selectedSpec,
  isCreating,
  issue,
  rollout,
  readonly,
});

const {
  shouldShow: shouldShowIsolationLevelSection,
  events: isolationLevelEvents,
} = provideIsolationLevelSettingContext({
  project,
  plan,
  selectedSpec,
  isCreating,
  issue,
  rollout,
  readonly,
});

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
  return (
    shouldShowTransactionModeSection.value ||
    shouldShowIsolationLevelSection.value ||
    shouldShowInstanceRoleSection.value ||
    shouldShowGhostSection.value ||
    shouldShowPreBackupSection.value
  );
});

transactionModeEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});

instanceRoleEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});

isolationLevelEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
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
