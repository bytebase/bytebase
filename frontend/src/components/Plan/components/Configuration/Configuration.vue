<template>
  <div class="flex flex-col gap-y-3 py-3">
    <h3 class="text-base">{{ $t("plan.options.self") }}</h3>
    <div v-if="shouldShow" class="flex flex-col gap-y-3">
      <InstanceRoleSection v-if="shouldShowInstanceRoleSection" />
      <TransactionModeSection v-if="shouldShowTransactionModeSection" />
      <IsolationLevelSection v-if="shouldShowIsolationLevelSection" />
      <PreBackupSection v-if="shouldShowPreBackupSection" />
      <GhostSection v-if="shouldShowGhostSection" />
    </div>
    <div v-else class="text-sm text-control-light">
      {{ $t("plan.options.no-options-available") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
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

const { isCreating, plan, events, issue, readonly, allowEdit, project } =
  usePlanContext();
const { selectedSpec } = useSelectedSpec();

const allowChange = computed(() => {
  // If readonly mode, disallow changes
  if (readonly?.value) {
    return false;
  }

  // Allow changes when creating
  if (isCreating.value) {
    return true;
  }

  // Disallow changes if the plan has started rollout.
  if (plan.value.hasRollout) {
    return false;
  }

  // If issue is not open, disallow
  if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  return allowEdit.value;
});

const providerArgs = {
  project,
  plan,
  selectedSpec,
  isCreating,
  allowChange,
};

const { shouldShow: shouldShowTransactionModeSection, events: txEvents } =
  provideTransactionModeSettingContext(providerArgs);
const { shouldShow: shouldShowInstanceRoleSection, events: roleEvents } =
  provideInstanceRoleSettingContext(providerArgs);
const { shouldShow: shouldShowIsolationLevelSection, events: isoEvents } =
  provideIsolationLevelSettingContext(providerArgs);
const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext(providerArgs);
const { shouldShow: shouldShowPreBackupSection, events: backupEvents } =
  providePreBackupSettingContext(providerArgs);

const shouldShow = computed(
  () =>
    shouldShowTransactionModeSection.value ||
    shouldShowIsolationLevelSection.value ||
    shouldShowInstanceRoleSection.value ||
    shouldShowGhostSection.value ||
    shouldShowPreBackupSection.value
);

// Forward all setting updates to plan context
const emitStatusChanged = () => events.emit("status-changed", { eager: true });
[txEvents, roleEvents, isoEvents, ghostEvents, backupEvents].forEach((e) =>
  e.on("update", emitStatusChanged)
);
</script>
