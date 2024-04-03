<template>
  <div class="w-full flex flex-col">
    <div class="w-full flex flex-row justify-between items-center mb-1">
      <span class="textlabel mr-4">{{ $t("issue.data-export.options") }}</span>
      <div
        v-if="!isCreating"
        class="flex flex-row justify-end items-center gap-2"
      >
        <NTooltip
          v-if="!state.isEditing"
          :disabled="denyEditTaskReasons.length === 0"
        >
          <template #trigger>
            <NButton
              size="tiny"
              tag="div"
              :disabled="denyEditTaskReasons.length > 0"
              @click="state.isEditing = true"
            >
              {{ $t("common.edit") }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="denyEditTaskReasons" />
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
        $t("issue.data-export.format")
      }}</span>
      <ExportFormatSelector
        :key="refreshKey"
        v-model:format="state.config.format"
        :editable="optionsEditable"
      />
    </div>
    <div class="w-full h-8 flex flex-row justify-start items-center">
      <span class="textinfolabel inline-block mr-2 !min-w-[64px]">{{
        $t("issue.data-export.encrypt")
      }}</span>
      <ExportPasswordInputer
        :key="refreshKey"
        v-model:password="state.config.password"
        :editable="optionsEditable"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, head } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, watch, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  allowUserToEditStatementForTask,
  notifyNotEditableLegacyIssue,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import {
  Plan_Spec,
  Plan_ExportDataConfig,
} from "@/types/proto/v1/rollout_service";
import ExportFormatSelector from "./ExportFormatSelector.vue";
import ExportPasswordInputer from "./ExportPasswordInputer.vue";

interface LocalState {
  config: Plan_ExportDataConfig;
  isEditing: boolean;
}

const { t } = useI18n();
const { issue, isCreating, selectedTask, events } = useIssueContext();
const currentUser = useCurrentUserV1();
const refreshKey = ref(0);

const spec = computed(
  () =>
    head(issue.value.planEntity?.steps.flatMap((step) => step.specs)) ||
    Plan_Spec.fromPartial({})
);

const state = reactive<LocalState>({
  config: Plan_ExportDataConfig.fromPartial({ ...spec.value.exportDataConfig }),
  isEditing: false,
});

const denyEditTaskReasons = computed(() => {
  return allowUserToEditStatementForTask(
    issue.value,
    selectedTask.value,
    currentUser.value
  );
});

const optionsEditable = computed(() => {
  return isCreating.value || state.isEditing;
});

const handleCancelEdit = () => {
  state.isEditing = false;
  state.config = Plan_ExportDataConfig.fromPartial({
    ...spec.value.exportDataConfig,
  });
  // Trigger a re-render of the child components.
  refreshKey.value++;
};

const handleSaveEdit = async () => {
  const planPatch = cloneDeep(issue.value.planEntity);
  if (!planPatch) {
    notifyNotEditableLegacyIssue();
    return;
  }

  const distinctSpecIds = new Set([spec.value.id]);
  const specsToPatch = planPatch.steps
    .flatMap((step) => step.specs)
    .filter((spec) => distinctSpecIds.has(spec.id));
  for (let i = 0; i < specsToPatch.length; i++) {
    const spec = specsToPatch[i];
    const config = spec.exportDataConfig;
    if (!config) continue;
    config.format = state.config.format;
    config.password = state.config.password || undefined;
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
  () => state.config,
  () => {
    if (!isCreating.value) {
      return;
    }
    spec.value.exportDataConfig = Plan_ExportDataConfig.fromPartial({
      ...spec.value.exportDataConfig,
      format: state.config.format,
      password: state.config.password,
    });
  },
  { deep: true }
);
</script>
