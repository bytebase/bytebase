<template>
  <div class="h-full relative flex flex-col gap-y-2 overflow-y-hidden text-sm">
    <MaskSpinner v-if="state.isRebasing" :zindexable="false" />

    <RebaseBranchSelect
      :project="project"
      :source-type="state.sourceType"
      :head-branch="headBranch"
      :source-branch="sourceBranch"
      :source-database="sourceDatabase"
      @update:source-type="state.sourceType = $event"
      @update:head-branch-name="handleUpdateHeadBranch"
      @update:source-branch-name="state.sourceBranchName = $event || null"
      @update:source-database-uid="state.sourceDatabaseUID = $event || null"
    />

    <RebaseBranchValidationStateView
      :project="project"
      :source-type="state.sourceType"
      :head-branch="headBranch"
      :source-branch="sourceBranch"
      :source-database="sourceDatabase"
      :is-loading-head-branch="isLoadingHeadBranch"
      :is-loading-source-branch="isLoadingSourceBranch"
      :is-validating="isValidating"
      :validation-state="validationState"
    />

    <div class="flex-1 flex flex-col overflow-y-hidden">
      <BranchComparison
        v-if="validationState?.branch"
        :project="project"
        :base="headBranch"
        :head="validationState?.branch"
        :is-base-loading="isLoadingHeadBranch"
        :is-head-loading="isValidating"
      >
        <template v-if="headBranch" #baseline-title>
          {{ headBranch.branchId }}
          {{ $t("branch.merge-rebase.before-rebase") }}
        </template>
        <template v-if="headBranch" #head-title>
          {{ headBranch.branchId }}
          {{ $t("branch.merge-rebase.after-rebase") }}
        </template>
      </BranchComparison>
      <ResolveConflict
        v-else-if="validationState"
        ref="resolveConflictRef"
        :project="project"
        :validation-state="validationState"
      />
    </div>

    <div class="flex flex-row justify-end items-center gap-x-3">
      <NButton
        type="primary"
        :disabled="!allowConfirm || state.isRebasing"
        @click="handleRebaseBranch"
      >
        {{ $t("common.confirm") }}
      </NButton>
    </div>
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
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useBranchStore, useDatabaseV1Store } from "@/store";
import { ComposedProject, UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { defer } from "@/utils";
import RebaseBranchSelect from "./RebaseBranchSelect.vue";
import RebaseBranchValidationStateView from "./RebaseBranchValidationStateView.vue";
import ResolveConflict from "./ResolveConflict.vue";
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
const resolveConflictRef = ref<InstanceType<typeof ResolveConflict>>();
const branchStore = useBranchStore();
const isLoadingHeadBranch = ref(false);
const isLoadingSourceBranch = ref(false);
const isValidating = ref(false);

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

const allowConfirm = computed(() => {
  return (
    headBranch.value !== undefined &&
    sourceBranchOrDatabase.value !== undefined &&
    validationState.value !== undefined
  );
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

  let mergedSchema = "";
  const resolveConflict = resolveConflictRef.value;
  if (resolveConflict) {
    const validation = resolveConflict.validateConflictSchema();
    if (!validation.valid) {
      const confirmed = await confirmRebaseWithMaybeConflict();
      if (!confirmed) return;
    }
    mergedSchema = validation.schema ?? "";
  }
  state.isRebasing = true;

  try {
    const response = await branchStore.rebaseBranch({
      name: head.name,
      sourceBranch: state.sourceType === "BRANCH" ? source.name : "",
      sourceDatabase: state.sourceType === "DATABASE" ? source.name : "",
      mergedSchema,
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
  (head, prevValue) => {
    if (!head) return;
    if (!prevValue && route.query.source) {
      // Initial head branch loaded
      return;
    }
    // Automatically set the sourceDatabase to the branch's baselineDatabase
    // if sourceDatabase is empty
    if (!state.sourceDatabaseUID) {
      const db = useDatabaseV1Store().getDatabaseByName(head.baselineDatabase);
      state.sourceDatabaseUID = db.uid;
      state.sourceType = "DATABASE";
    }

    // Automatically select the branch's parent as rebase source branch
    // if rebase source is empty
    if (head.parentBranch && !state.sourceBranchName) {
      state.sourceBranchName = head.parentBranch;
      state.sourceType = "BRANCH";
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
    state.sourceType = "BRANCH";
  }
});
</script>
