<template>
  <div class="relative">
    <div v-if="icon === 'system'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <img class="mt-1" src="@/assets/logo-icon.svg" alt="Bytebase" />
      </div>
    </div>
    <div v-else-if="icon === 'avatar'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PrincipalAvatar
          :user="user"
          override-class="w-7 h-7 font-medium"
          override-text-size="0.8rem"
        />
      </div>
    </div>
    <div v-else-if="icon === 'create'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-solid:plus-sm class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'update'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-solid:pencil class="w-4 h-4 text-control" />
      </div>
    </div>
    <div
      v-else-if="icon === 'run' || icon === 'rollout'"
      class="relative pl-0.5"
    >
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:play class="w-6 h-6 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'approve-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:thumb-up class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'reject-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-warning rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons:pause-solid class="w-5 h-5 text-white" />
      </div>
    </div>
    <div v-else-if="icon === 're-request-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-success rounded-full ring-4 ring-white flex items-center justify-center icon-re-request-review"
      >
        <heroicons:play class="w-4 h-4 text-white ml-px" />
      </div>
    </div>
    <div v-else-if="icon === 'cancel'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:minus class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'fail'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:exclamation-circle class="w-6 h-6 text-error" />
      </div>
    </div>
    <div v-else-if="icon === 'complete'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:check-circle class="w-6 h-6 text-success" />
      </div>
    </div>
    <div v-else-if="icon === 'skip'" class="relative pl-1">
      <div
        class="w-6 h-6 bg-gray-200 rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <SkipIcon class="w-5 h-5 text-gray-500" />
      </div>
    </div>
    <div v-else-if="icon === 'commit'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:code class="w-5 h-5 text-control" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import PrincipalAvatar from "@/components/PrincipalAvatar.vue";
import {
  IssueCommentType,
  useUserStore,
  type ComposedIssueComment,
} from "@/store";
import { extractUserId } from "@/store";
import {
  IssueComment_Approval,
  IssueComment_Approval_Status,
  IssueComment_IssueUpdate,
  IssueComment_TaskPriorBackup,
  IssueComment_TaskUpdate,
  IssueComment_TaskUpdate_Status,
} from "@/types/proto/v1/issue_service";

type ActionIconType =
  | "avatar"
  | "system"
  | "create"
  | "update"
  | "run"
  | "approve-review"
  | "reject-review"
  | "re-request-review"
  | "rollout"
  | "cancel"
  | "fail"
  | "complete"
  | "skip"
  | "commit";

const props = defineProps<{
  issueComment: ComposedIssueComment;
}>();

const userStore = useUserStore();

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier(props.issueComment.creator);
});

const icon = computed((): ActionIconType => {
  const { issueComment } = props;
  if (issueComment.type === IssueCommentType.APPROVAL) {
    const { status } = IssueComment_Approval.fromPartial(
      issueComment.approval || {}
    );
    switch (status) {
      case IssueComment_Approval_Status.APPROVED:
        return "approve-review";
      case IssueComment_Approval_Status.REJECTED:
        return "reject-review";
      case IssueComment_Approval_Status.PENDING:
        return "re-request-review";
    }
  } else if (issueComment.type === IssueCommentType.STAGE_END) {
    return "complete";
  } else if (issueComment.type === IssueCommentType.TASK_UPDATE) {
    const { toStatus } = IssueComment_TaskUpdate.fromPartial(
      issueComment.taskUpdate || {}
    );
    let action: ActionIconType = "update";
    if (toStatus !== undefined) {
      switch (toStatus) {
        case IssueComment_TaskUpdate_Status.PENDING: {
          action = "rollout";
          break;
        }
        case IssueComment_TaskUpdate_Status.RUNNING: {
          action = "run";
          break;
        }
        case IssueComment_TaskUpdate_Status.DONE: {
          action = "complete";
          break;
        }
        case IssueComment_TaskUpdate_Status.FAILED: {
          action = "fail";
          break;
        }
        case IssueComment_TaskUpdate_Status.SKIPPED: {
          action = "skip";
          break;
        }
        case IssueComment_TaskUpdate_Status.CANCELED: {
          action = "cancel";
          break;
        }
      }
    }
    return action;
  } else if (issueComment.type === IssueCommentType.ISSUE_UPDATE) {
    const { toTitle, toDescription, toLabels, fromLabels } =
      IssueComment_IssueUpdate.fromPartial(issueComment.issueUpdate || {});
    if (
      toTitle !== undefined ||
      toDescription !== undefined ||
      toLabels.length !== 0 ||
      fromLabels.length !== 0
    ) {
      return "update";
    }
    // Otherwise, show avatar icon based on the creator.
  } else if (issueComment.type === IssueCommentType.TASK_PRIOR_BACKUP) {
    const taskPriorBackup = IssueComment_TaskPriorBackup.fromPartial(
      issueComment.taskPriorBackup || {}
    );
    if (taskPriorBackup.error !== "") {
      return "fail";
    } else {
      return "complete";
    }
  }

  return extractUserId(issueComment.creator) == userStore.systemBotUser?.email
    ? "system"
    : "avatar";
});
</script>

<style scoped>
.icon-re-request-review :deep(path) {
  stroke-width: 3 !important;
}
</style>
