<template>
  <div class="w-full flex flex-col">
    <div class="w-full flex flex-row justify-between items-center mb-1">
      <span class="textlabel mr-4">{{ $t("issue.data-export.options") }}</span>
      <div
        v-if="showEditButtons"
        class="flex flex-row justify-end items-center gap-2"
      >
        <NTooltip
          v-if="!state.isEditing"
          :disabled="denyEditReasons.length === 0"
        >
          <template #trigger>
            <NButton
              size="tiny"
              tag="div"
              :disabled="denyEditReasons.length > 0"
              @click="state.isEditing = true"
            >
              {{ $t("common.edit") }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="denyEditReasons" />
          </template>
        </NTooltip>
        <template v-else>
          <NButton size="tiny" @click="handleSaveEdit">
            {{ $t("common.save") }}
          </NButton>
          <NButton size="tiny" quaternary @click.prevent="handleCancelEdit">
            {{ $t("common.cancel") }}
          </NButton>
        </template>
      </div>
    </div>
    <div class="w-full h-8 flex flex-row justify-start items-center">
      <span class="textinfolabel inline-block mr-2 !min-w-[64px]">{{
        $t("issue.sql-review.statement-type")
      }}</span>
      <StatementTypeSelector
        :key="refreshKey"
        v-model:statement-type="state.statementType"
        :editable="optionsEditable"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, watch, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  notifyNotEditableLegacyIssue,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_ChangeDatabaseConfig,
} from "@/types/proto/v1/rollout_service";
import { hasProjectPermissionV2 } from "@/utils";
import StatementTypeSelector from "./StatementTypeSelector.vue";
import type { StatementType } from "./types";

interface LocalState {
  statementType: StatementType;
  isEditing: boolean;
}

const { t } = useI18n();
const { issue, isCreating, selectedSpec, events } = useIssueContext();
const currentUser = useCurrentUserV1();
const refreshKey = ref(0);

const state = reactive<LocalState>({
  statementType:
    (selectedSpec.value.changeDatabaseConfig?.type as StatementType) ||
    Plan_ChangeDatabaseConfig_Type.MIGRATE,
  isEditing: false,
});

const showEditButtons = computed(() => {
  return !isCreating.value && issue.value.status === IssueStatus.OPEN;
});

const optionsEditable = computed(() => {
  return isCreating.value || (showEditButtons.value && state.isEditing);
});

const denyEditReasons = computed(() => {
  const reasons: string[] = [];
  if (
    !hasProjectPermissionV2(
      issue.value.projectEntity,
      currentUser.value,
      "bb.plans.update"
    )
  ) {
    reasons.push("Permission denied");
  }
  return reasons;
});

const handleCancelEdit = () => {
  state.isEditing = false;
  state.statementType =
    (selectedSpec.value.changeDatabaseConfig?.type as StatementType) ||
    Plan_ChangeDatabaseConfig_Type.MIGRATE;
  // Trigger a re-render of the child components.
  refreshKey.value++;
};

const handleSaveEdit = async () => {
  const planPatch = cloneDeep(issue.value.planEntity);
  if (!planPatch) {
    notifyNotEditableLegacyIssue();
    return;
  }

  const distinctSpecIds = new Set([selectedSpec.value.id]);
  const specsToPatch = planPatch.steps
    .flatMap((step) => step.specs)
    .filter((spec) => distinctSpecIds.has(spec.id));
  for (let i = 0; i < specsToPatch.length; i++) {
    const spec = specsToPatch[i];
    const config = spec.changeDatabaseConfig;
    if (!config) continue;
    config.type = state.statementType;
  }

  const updatedPlan = await rolloutServiceClient.updatePlan({
    plan: planPatch,
    updateMask: ["steps"],
  });
  issue.value.planEntity = updatedPlan;

  events.emit("status-changed", { eager: true });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  handleCancelEdit();
};

watch(
  () => state.statementType,
  () => {
    if (!isCreating.value) {
      return;
    }
    selectedSpec.value.changeDatabaseConfig =
      Plan_ChangeDatabaseConfig.fromPartial({
        ...selectedSpec.value.changeDatabaseConfig,
        type: state.statementType,
      });
  },
  { deep: true }
);
</script>
