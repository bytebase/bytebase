<template>
  <div v-if="shouldShow" class="flex flex-col gap-1">
    <span class="text-base">{{ $t("plan.options.self") }}</span>
    <div class="flex flex-wrap items-center gap-x-4 sm:gap-x-6 gap-y-1">
      <div v-if="shouldShowInstanceRoleSection" class="flex items-center gap-1">
        <span class="text-sm text-control-light">{{ $t("common.role.self") }}</span>
        <InstanceRoleSelect />
      </div>
      <div v-if="shouldShowTransactionModeSection" class="flex items-center gap-1">
        <span class="text-sm text-control-light">{{
          $t("issue.transaction-mode.label")
        }}</span>
        <TransactionModeSwitch />
      </div>
      <div v-if="shouldShowIsolationLevelSection" class="flex items-center gap-1">
        <span class="text-sm text-control-light">{{
          $t("plan.isolation-level.self")
        }}</span>
        <IsolationLevelSelect />
      </div>
      <div v-if="shouldShowPreBackupSection" class="flex items-center gap-1">
        <span class="text-sm text-control-light">{{ $t("task.prior-backup") }}</span>
        <PreBackupSwitch />
      </div>
      <div v-if="shouldShowGhostSection" class="flex items-center gap-1">
        <span class="text-sm text-control-light">{{
          $t("task.online-migration.self")
        }}</span>
        <GhostSwitch />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanContext } from "../../../logic";
import { provideGhostSettingContext } from "../../Configuration/GhostSection/context";
import GhostSwitch from "../../Configuration/GhostSection/GhostSwitch.vue";
import { provideInstanceRoleSettingContext } from "../../Configuration/InstanceRoleSection/context";
import InstanceRoleSelect from "../../Configuration/InstanceRoleSection/InstanceRoleSelect.vue";
import { provideIsolationLevelSettingContext } from "../../Configuration/IsolationLevelSection/context";
import IsolationLevelSelect from "../../Configuration/IsolationLevelSection/IsolationLevelSelect.vue";
import { providePreBackupSettingContext } from "../../Configuration/PreBackupSection/context";
import PreBackupSwitch from "../../Configuration/PreBackupSection/PreBackupSwitch.vue";
import { provideTransactionModeSettingContext } from "../../Configuration/TransactionModeSection/context";
import TransactionModeSwitch from "../../Configuration/TransactionModeSection/TransactionModeSwitch.vue";
import { useSelectedSpec } from "../../SpecDetailView/context";

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
  issue,
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
