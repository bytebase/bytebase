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
      @finish="handleMergeBranch"
    >
      <template #0>
        <SelectBranchStep
          v-model:delete-branch-after-merged="state.deleteBranchAfterMerged"
          :project="project"
          :head-branch="headBranch"
          :target-branch="targetBranch"
          :is-loading-head-branch="isLoadingHeadBranch"
          :is-loading-target-branch="isLoadingTargetBranch"
          :is-validating="isValidating"
          :validation-state="validationState"
          @update:head-branch-name="handleUpdateHeadBranch"
          @update:target-branch-name="state.targetBranchName = $event || null"
        />
      </template>
      <template #1>
        <MergeBranchStep
          v-if="targetBranch && headBranch && validationState?.branch"
          :project="project"
          :merged-branch="validationState.branch"
        />
      </template>
    </StepTab>
    <MaskSpinner v-if="state.isMerging" />
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { Status } from "nice-grpc-common";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { StepTab } from "@/components/v2";
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useBranchStore } from "@/store";
import { ComposedProject } from "@/types";
import { getErrorCode } from "@/utils/grpcweb";
import MergeBranchStep from "./MergeBranchStep.vue";
import SelectBranchStep from "./SelectBranchStep.vue";
import { MergeBranchValidationState } from "./types";

interface LocalState {
  currentStepIndex: number;
  targetBranchName: string | null;
  deleteBranchAfterMerged: boolean;
  isMerging: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  headBranchName: string | null;
}>();

const emit = defineEmits<{
  (event: "merged", headBranchName: string, targetBranchName: string): void;
  (event: "update:head-branch-name", branchName: string | null): void;
}>();

const state = reactive<LocalState>({
  currentStepIndex: 0,
  targetBranchName: null,
  deleteBranchAfterMerged: false,
  isMerging: false,
});
const { t } = useI18n();
const branchStore = useBranchStore();
const isLoadingHeadBranch = ref(false);
const isLoadingTargetBranch = ref(false);
const isValidating = ref(false);
const combinedIsLoadingBranch = computed(() => {
  return isLoadingHeadBranch.value || isLoadingTargetBranch.value;
});

const stepList = computed(() => [
  { title: t("branch.select-branch") },
  { title: t("common.preview") },
]);
const STEP_SELECT_BRANCH = 0;
const STEP_MERGE_BRANCH = 1;

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

const targetBranch = computedAsync(
  async () => {
    const name = state.targetBranchName;
    if (!name) {
      return undefined;
    }
    // Don't use local store cache since we need to ensure the branch is
    // fresh clean here
    return await branchStore.fetchBranchByName(name, /* !useCache */ false);
  },
  undefined,
  {
    evaluating: isLoadingTargetBranch,
  }
);

const validationState = computedAsync(
  async (): Promise<MergeBranchValidationState | undefined> => {
    const head = headBranch.value;
    const target = targetBranch.value;
    if (!head) return;
    if (!target) return;
    try {
      const branch = await branchServiceClient.mergeBranch(
        {
          name: target.name,
          headBranch: head.name,
          validateOnly: true,
        },
        {
          silent: true,
        }
      );
      return {
        status: Status.OK,
        branch,
      };
    } catch (err) {
      const status = getErrorCode(err);
      return {
        status,
      };
    }
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
      targetBranch.value !== undefined &&
      validationState.value?.status === Status.OK
    );
  }
  if (state.currentStepIndex === STEP_MERGE_BRANCH) {
    return true;
  }

  console.error("[BranchMergeView.allowNextStep] should never reach this line");
  return false;
});

const handleMergeBranch = async () => {
  const target = targetBranch.value;
  if (!target) return;
  const head = headBranch.value;
  if (!head) return;
  state.isMerging = true;

  try {
    await branchStore.mergeBranch({
      name: target.name,
      headBranch: head.name,
      etag: "",
      validateOnly: false,
    });
    if (state.deleteBranchAfterMerged) {
      await branchStore.deleteBranch(head.name);
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.merge-to-main-successfully"),
    });

    emit("merged", head.name, target.name);
  } catch (error: any) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: error.details,
    });
  } finally {
    state.isMerging = false;
  }
};
</script>
