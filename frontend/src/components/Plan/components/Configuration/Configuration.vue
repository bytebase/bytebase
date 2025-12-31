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
const { isCreating, plan, events, issue, readonly } = usePlanContext();
const { selectedSpec } = useSelectedSpec();

const providerArgs = {
  project,
  plan,
  selectedSpec,
  isCreating,
  issue,
  readonly,
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
