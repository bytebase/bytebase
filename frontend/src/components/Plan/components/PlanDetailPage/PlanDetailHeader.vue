<template>
  <div class="py-2 px-2 sm:px-4">
    <div class="flex flex-row items-center justify-between gap-2">
      <!-- Status tags (closed/done) -->
      <NTag v-if="isClosed" round type="default">
        <template #icon>
          <BanIcon class="w-4 h-4" />
        </template>
        {{ $t("common.closed") }}
      </NTag>

      <!-- Title -->
      <TitleInput />

      <!-- Actions -->
      <div class="flex items-center gap-x-2">
        <!-- Create plan (when creating) -->
        <CreateButton v-if="isCreating" />

        <!-- Submit for Review (plan exists, no issue yet) -->
        <template v-else>
          <CreateIssueButton
            v-if="showSubmitForReview"
            :disabled="submitDisabled"
            :disabled-reason="submitDisabledReason"
          />

          <!-- Close / Reopen plan (no issue, no rollout) -->
          <NButton
            v-if="showClosePlan"
            size="medium"
            @click="handleClosePlan"
          >
            {{ $t("common.close") }}
          </NButton>
          <NButton
            v-if="showReopenPlan"
            size="medium"
            @click="handleReopenPlan"
          >
            {{ $t("common.reopen") }}
          </NButton>
        </template>

        <!-- Sidebar toggle for small screens -->
        <NButton
          v-if="showSidebarToggle"
          quaternary
          size="medium"
          @click="mobileSidebarOpen = true"
        >
          <template #icon>
            <MenuIcon class="w-5 h-5" />
          </template>
        </NButton>
      </div>
    </div>
    <DescriptionSection v-if="showDescription" />
  </div>
</template>

<script setup lang="ts">
import { clone, create } from "@bufbuild/protobuf";
import { BanIcon, MenuIcon } from "lucide-vue-next";
import { NButton, NTag, useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { planServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { isValidPlanName } from "@/utils";
import { emitPlanStatusChanged, usePlanContext } from "../../logic";
import { useSidebarContext } from "../../logic/sidebar";
import { useEditorState } from "../../logic/useEditorState";
import {
  CreateButton,
  CreateIssueButton,
} from "../HeaderSection/Actions/create";
import DescriptionSection from "../HeaderSection/DescriptionSection.vue";
import TitleInput from "../HeaderSection/TitleInput.vue";

const { t } = useI18n();
const dialog = useDialog();

const { isCreating, plan, events } = usePlanContext();
const editorState = useEditorState();
const { mode: sidebarMode, mobileSidebarOpen } = useSidebarContext();

const isClosed = computed(() => {
  return plan.value.state === State.DELETED;
});

const showDescription = computed(() => {
  if (isCreating.value) return true;
  return isValidPlanName(plan.value.name);
});

const showSidebarToggle = computed(() => {
  return sidebarMode.value === "MOBILE";
});

// Submit for Review: visible when plan exists, no issue, plan is active
const showSubmitForReview = computed(() => {
  return (
    isValidPlanName(plan.value.name) &&
    !plan.value.issue &&
    plan.value.state === State.ACTIVE
  );
});

const submitDisabled = computed(() => editorState.isEditing.value);
const submitDisabledReason = computed(() =>
  submitDisabled.value
    ? t("plan.editor.save-changes-before-continuing")
    : undefined
);

// Close plan: visible when plan exists, active, no issue/rollout
const showClosePlan = computed(() => {
  return (
    isValidPlanName(plan.value.name) &&
    !plan.value.issue &&
    !plan.value.hasRollout &&
    plan.value.state === State.ACTIVE
  );
});

// Reopen plan: visible when plan is closed, no issue/rollout
const showReopenPlan = computed(() => {
  return (
    isValidPlanName(plan.value.name) &&
    !plan.value.issue &&
    !plan.value.hasRollout &&
    plan.value.state === State.DELETED
  );
});

const updatePlanState = async (state: State) => {
  try {
    const planPatch = clone(PlanSchema, plan.value);
    planPatch.state = state;
    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["state"] },
    });
    const updated = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, updated);
    emitPlanStatusChanged(events);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: String(error),
    });
  }
};

const handleClosePlan = () => {
  dialog.warning({
    title: t("common.close"),
    content: t("plan.state.close-confirm"),
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: () => updatePlanState(State.DELETED),
  });
};

const handleReopenPlan = () => {
  dialog.info({
    title: t("common.reopen"),
    content: t("plan.state.reopen-confirm"),
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: () => updatePlanState(State.ACTIVE),
  });
};
</script>
