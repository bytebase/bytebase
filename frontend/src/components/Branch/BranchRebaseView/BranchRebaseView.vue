<template>
  <div class="h-full relative">
    <StepTab
      :step-list="stepList"
      :current-index="state.currentStepIndex"
      :show-cancel="false"
      :allow-next="allowNextStep"
      class="h-full flex flex-col !space-y-0 gap-y-4"
      pane-class="flex-1 flex flex-col gap-y-2 relative"
      footer-class="!space-y-0 !border-0 !pt-0"
      @update:current-index="tryChangeStep"
      @finish="handleRebaseBranch"
    >
      <template #0>
        <SelectBranchStep
          :project="project"
          :head-branch="headBranch"
          :source-branch="sourceBranch"
          :is-loading-head-branch="isLoadingHeadBranch"
          :is-loading-source-branch="isLoadingSourceBranch"
          :is-validating="isValidating"
          :validation-state="validationState"
          @update:head-branch-name="handleUpdateHeadBranch"
          @update:source-branch-name="state.sourceBranchName = $event || null"
        />
      </template>
      <template #1>
        <RebaseBranchStep
          v-if="sourceBranch && headBranch && validationState"
          :project="project"
          :validation-state="validationState"
        />
      </template>
    </StepTab>
    <MaskSpinner v-if="state.isRebasing" />
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { StepTab } from "@/components/v2";
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useBranchStore } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import RebaseBranchStep from "./RebaseBranchStep.vue";
import SelectBranchStep from "./SelectBranchStep.vue";
import { RebaseBranchValidationState } from "./types";

interface LocalState {
  currentStepIndex: number;
  sourceBranchName: string | null;
  isRebasing: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  headBranchName: string | null;
}>();

const emit = defineEmits<{
  (
    event: "rebased",
    rebasedBranch: Branch,
    headBranchName: string,
    headBranch: Branch | undefined
  ): void;
  (event: "update:head-branch-name", branchName: string | null): void;
}>();

const state = reactive<LocalState>({
  currentStepIndex: 0,
  sourceBranchName: null,
  isRebasing: false,
});
const { t } = useI18n();
const branchStore = useBranchStore();
const isLoadingHeadBranch = ref(false);
const isLoadingSourceBranch = ref(false);
const isValidating = ref(false);
const combinedIsLoadingBranch = computed(() => {
  return isLoadingHeadBranch.value || isLoadingSourceBranch.value;
});

const stepList = computed(() => [
  { title: t("branch.select-branch") },
  { title: t("common.preview") },
]);
const STEP_SELECT_BRANCH = 0;
const STEP_REBASE_BRANCH = 1;

const headBranch = computedAsync(
  async () => {
    const name = props.headBranchName;
    if (!name) {
      return undefined;
    }
    // Don't use local store cache since we need to ensure the branch is
    // fresh clean here
    return await branchStore.fetchBranchByName(name, /* !useCache */ false);
  },
  undefined,
  {
    evaluating: isLoadingHeadBranch,
  }
);

const sourceBranch = computedAsync(
  async () => {
    const name = state.sourceBranchName;
    if (!name) {
      return undefined;
    }
    // Don't use local store cache since we need to ensure the branch is
    // fresh clean here
    return await branchStore.fetchBranchByName(name, /* !useCache */ false);
  },
  undefined,
  {
    evaluating: isLoadingSourceBranch,
  }
);

const validationState = computedAsync(
  async (): Promise<RebaseBranchValidationState | undefined> => {
    const head = headBranch.value;
    const source = sourceBranch.value;
    if (!head) return;
    if (!source) return;
    const response = await branchServiceClient.rebaseBranch({
      name: head.name,
      sourceBranch: source.name,
    });
    return response;
  },
  undefined,
  {
    evaluating: isValidating,
  }
);
const handleUpdateHeadBranch = (branchName: string | undefined) => {
  emit("update:head-branch-name", branchName || null);
};

const tryChangeStep = (nextStepIndex: number) => {
  if (combinedIsLoadingBranch.value) {
    return;
  }
  state.currentStepIndex = nextStepIndex;
};

const allowNextStep = computed(() => {
  if (state.currentStepIndex === STEP_SELECT_BRANCH) {
    return (
      headBranch.value !== undefined &&
      sourceBranch.value !== undefined &&
      validationState.value !== undefined
    );
  }
  if (state.currentStepIndex === STEP_REBASE_BRANCH) {
    return true;
  }

  console.error(
    "[BranchRebaseView.allowNextStep] should never reach this line"
  );
  return false;
});

const handleRebaseBranch = async () => {
  const target = sourceBranch.value;
  if (!target) return;
  const head = headBranch.value;
  if (!head) return;
  state.isRebasing = true;

  try {
    const rebasedBranch = await branchStore.mergeBranch({
      name: target.name,
      headBranch: head.name,
      etag: "",
      validateOnly: false,
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("branch.merge-rebase.rebase-succeeded"),
    });

    emit("rebased", rebasedBranch, head.name, head);
  } catch (error: any) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: error.details,
    });
  } finally {
    state.isRebasing = false;
  }
};
</script>
