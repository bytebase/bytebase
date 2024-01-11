<template>
  <div class="h-full relative">
    <StepTab
      :step-list="stepList"
      :current-index="state.currentStepIndex"
      :show-cancel="false"
      :allow-next="allowNextStep"
      class="h-full flex flex-col !space-y-0 gap-y-2"
      pane-class="flex-1 flex flex-col gap-y-2 relative overflow-hidden"
      footer-class="!space-y-0 !border-0 !pt-0"
      @update:current-index="tryChangeStep"
      @finish="handleRebaseBranch"
    >
      <template #0>
        <SelectBranchStep
          :project="project"
          :source-type="state.sourceType"
          :head-branch="headBranch"
          :source-branch="sourceBranch"
          :source-database="sourceDatabase"
          :is-loading-head-branch="isLoadingHeadBranch"
          :is-loading-source-branch="isLoadingSourceBranch"
          :is-validating="isValidating"
          :validation-state="validationState"
          @update:source-type="state.sourceType = $event"
          @update:head-branch-name="handleUpdateHeadBranch"
          @update:source-branch-name="state.sourceBranchName = $event || null"
          @update:source-database-uid="state.sourceDatabaseUID = $event || null"
        />
      </template>
      <template #1>
        <RebaseBranchStep
          v-if="headBranch && sourceBranchOrDatabase && validationState"
          ref="rebaseBranchStepRef"
          :project="project"
          :validation-state="validationState"
        />
      </template>
    </StepTab>
    <MaskSpinner v-if="state.isRebasing" :zindexable="false" />
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { useDialog } from "naive-ui";
import { ClientError, Status } from "nice-grpc-common";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { StepTab } from "@/components/v2";
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useBranchStore, useDatabaseV1Store } from "@/store";
import { ComposedProject, UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { defer } from "@/utils";
import RebaseBranchStep from "./RebaseBranchStep.vue";
import SelectBranchStep from "./SelectBranchStep.vue";
import { RebaseBranchValidationState, RebaseSourceType } from "./types";

interface LocalState {
  currentStepIndex: number;
  sourceBranchName: string | null;
  sourceDatabaseUID: string | null;
  sourceType: RebaseSourceType;
  isRebasing: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  headBranchName: string | null;
}>();

const emit = defineEmits<{
  (event: "rebased", rebasedBranch: Branch): void;
  (event: "update:head-branch-name", branchName: string | null): void;
}>();

const state = reactive<LocalState>({
  currentStepIndex: 0,
  sourceBranchName: null,
  sourceDatabaseUID: null,
  sourceType: "BRANCH",
  isRebasing: false,
});
const { t } = useI18n();
const route = useRoute();
const $dialog = useDialog();
const rebaseBranchStepRef = ref<InstanceType<typeof RebaseBranchStep>>();
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
    if (state.sourceType === "DATABASE") {
      return undefined;
    }
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
const sourceDatabase = computed(() => {
  if (state.sourceType === "BRANCH") {
    return undefined;
  }
  const uid = state.sourceDatabaseUID;
  if (!uid || uid === String(UNKNOWN_ID)) {
    return undefined;
  }
  return useDatabaseV1Store().getDatabaseByUID(uid);
});
const sourceBranchOrDatabase = computed(() => {
  if (state.sourceType === "BRANCH") {
    return sourceBranch.value;
  }
  return sourceDatabase.value;
});

const validationState = computedAsync(
  async (): Promise<RebaseBranchValidationState | undefined> => {
    const head = headBranch.value;
    const source = sourceBranchOrDatabase.value;
    if (!head) return;
    if (!source) return;
    const response = await branchServiceClient.rebaseBranch({
      name: head.name,
      sourceBranch: state.sourceType === "BRANCH" ? source.name : undefined,
      sourceDatabase: state.sourceType === "DATABASE" ? source.name : undefined,
      validateOnly: true,
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
      sourceBranchOrDatabase.value !== undefined &&
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

const confirmRebaseWithMaybeConflict = () => {
  const d = defer<boolean>();
  $dialog.warning({
    title: t("branch.merge-rebase.confirm-rebase"),
    content: t("branch.merge-rebase.conflict-not-resolved"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onNegativeClick: () => {
      d.resolve(false);
    },
    onPositiveClick: () => {
      d.resolve(true);
    },
  });
  return d.promise;
};

const handleRebaseBranch = async () => {
  const source = sourceBranchOrDatabase.value;
  if (!source) return;
  const head = headBranch.value;
  if (!head) return;
  const rebaseBranchStep = rebaseBranchStepRef.value;
  if (!rebaseBranchStep) return;
  state.isRebasing = true;

  try {
    const validation = rebaseBranchStep.validateConflictSchema();
    if (!validation.valid) {
      const confirmed = await confirmRebaseWithMaybeConflict();
      if (!confirmed) return;
    }
    const response = await branchStore.rebaseBranch({
      name: head.name,
      sourceBranch: state.sourceType === "BRANCH" ? source.name : "",
      sourceDatabase: state.sourceType === "DATABASE" ? source.name : "",
      mergedSchema: validation.schema ?? "",
      etag: "",
      validateOnly: false,
    });
    if (!response.branch) {
      throw new ClientError(
        "BranchService.RebaseBranch",
        Status.ABORTED,
        "rebase failed"
      );
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("branch.merge-rebase.rebase-succeeded"),
    });

    emit("rebased", response.branch);
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

watch(
  headBranch,
  (head) => {
    if (head) {
      // Automatically set the sourceDatabase to the branch's baselineDatabase
      // if sourceDatabase is empty
      if (!state.sourceDatabaseUID) {
        const db = useDatabaseV1Store().getDatabaseByName(
          head.baselineDatabase
        );
        state.sourceDatabaseUID = db.uid;
        state.sourceType = "DATABASE";
      }

      // Automatically select the branch's parent as rebase source branch
      // if rebase source is empty
      if (head.parentBranch && !state.sourceBranchName) {
        state.sourceBranchName = head.parentBranch;
        state.sourceType = "BRANCH";
      }
    }
  },
  {
    immediate: true,
  }
);

onMounted(() => {
  if (route.query.source) {
    const source = `${props.project.name}/branches/${route.query.source}`;
    state.sourceBranchName = source;
  }
});
</script>
