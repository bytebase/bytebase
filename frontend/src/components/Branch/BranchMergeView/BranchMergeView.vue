<template>
  <div class="h-full relative flex flex-col gap-y-2 overflow-y-hidden text-sm">
    <MaskSpinner v-if="state.isMerging" />

    <MergeBranchSelect
      :project="project"
      :head-branch="headBranch"
      :target-branch="targetBranch"
      @update:head-branch-name="handleUpdateHeadBranch"
      @update:target-branch-name="state.targetBranchName = $event || null"
    />

    <MergeBranchValidationStateView
      :project="project"
      :head-branch="headBranch"
      :target-branch="targetBranch"
      :is-loading-head-branch="isLoadingHeadBranch"
      :is-loading-target-branch="isLoadingTargetBranch"
      :is-validating="isValidating"
      :validation-state="validationState"
    />

    <div class="flex-1">
      <BranchComparison
        v-if="validationState?.branch"
        :project="project"
        :base="targetBranch"
        :head="validationState?.branch"
        :is-base-loading="isLoadingTargetBranch"
        :is-head-loading="isValidating"
      >
        <template v-if="targetBranch" #baseline-title>
          {{ targetBranch.branchId }}
          {{ $t("branch.merge-rebase.before-merge") }}
        </template>
        <template v-if="targetBranch" #head-title>
          {{ targetBranch.branchId }}
          {{ $t("branch.merge-rebase.after-merge") }}
        </template>
      </BranchComparison>
      <div v-else-if="validationState?.errmsg" class="text-error text-sm">
        {{ validationState.errmsg }}
      </div>
    </div>

    <div class="flex flex-row justify-end items-center gap-x-3">
      <MergeBranchButton
        :button-props="{
          type: 'primary',
          disabled: !allowConfirm || state.isMerging,
        }"
        @perform-action="handleMergeBranch"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { Status } from "nice-grpc-common";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useBranchStore } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { extractGrpcErrorMessage, getErrorCode } from "@/utils/grpcweb";
import BranchComparison from "../common/BranchComparison.vue";
import MergeBranchButton from "./MergeBranchButton.vue";
import MergeBranchSelect from "./MergeBranchSelect.vue";
import MergeBranchValidationStateView from "./MergeBranchValidationStateView.vue";
import { MergeBranchValidationState, PostMergeAction } from "./types";

interface LocalState {
  targetBranchName: string | null;
  isMerging: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  headBranchName: string | null;
}>();

const emit = defineEmits<{
  (
    event: "merged",
    mergedBranch: Branch,
    headBranchName: string,
    headBranch: Branch | undefined
  ): void;
  (event: "update:head-branch-name", branchName: string | null): void;
}>();

const state = reactive<LocalState>({
  targetBranchName: null,
  isMerging: false,
});
const { t } = useI18n();
const branchStore = useBranchStore();
const isLoadingHeadBranch = ref(false);
const isLoadingTargetBranch = ref(false);
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
      const errmsg = extractGrpcErrorMessage(err);
      return {
        status,
        errmsg,
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

const allowConfirm = computed(() => {
  return (
    headBranch.value !== undefined &&
    targetBranch.value !== undefined &&
    validationState.value?.status === Status.OK
  );
});

const handleMergeBranch = async (post: PostMergeAction) => {
  const target = targetBranch.value;
  if (!target) return;
  const head = headBranch.value;
  if (!head) return;
  state.isMerging = true;

  try {
    const mergedBranch = await branchStore.mergeBranch({
      name: target.name,
      headBranch: head.name,
      etag: "",
      validateOnly: false,
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("branch.merge-rebase.merge-succeeded"),
    });
    if (post === "DELETE") {
      await branchStore.deleteBranch(head.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });

      emit("merged", mergedBranch, head.name, undefined);
      return;
    }
    if (post === "REBASE") {
      const response = await branchStore.rebaseBranch({
        name: head.name,
        sourceBranch: target.name,
        sourceDatabase: "",
        mergedSchema: "",
        etag: "",
        validateOnly: false,
      });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("branch.merge-rebase.rebase-succeeded"),
      });
      emit("merged", mergedBranch, head.name, response.branch);
      return;
    }
    emit("merged", mergedBranch, head.name, head);
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

watch(
  headBranch,
  (head) => {
    if (head) {
      // Automatically select the branch's parent as merge target branch
      // if merge target is empty
      if (head.parentBranch && !state.targetBranchName) {
        state.targetBranchName = head.parentBranch;
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
