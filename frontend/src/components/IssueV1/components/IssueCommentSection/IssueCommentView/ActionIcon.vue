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
        <UserAvatar
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
        <PlusIcon class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'update'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PencilIcon class="w-4 h-4 text-control" />
      </div>
    </div>
    <div
      v-else-if="icon === 'run' || icon === 'rollout'"
      class="relative pl-0.5"
    >
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PlayCircleIcon class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'approve-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-success rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <ThumbsUpIcon class="w-4 h-4 text-white" />
      </div>
    </div>
    <div v-else-if="icon === 'reject-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-warning rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PencilIcon class="w-4 h-4 text-white" />
      </div>
    </div>
    <div v-else-if="icon === 're-request-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center icon-re-request-review"
      >
        <PlayIcon class="w-4 h-4 text-control ml-px" />
      </div>
    </div>
    <div v-else-if="icon === 'cancel'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <MinusIcon class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon === 'fail'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <CircleAlertIcon class="w-5 h-5 text-error" />
      </div>
    </div>
    <div v-else-if="icon === 'complete'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <CheckCircle2Icon class="w-6 h-6 text-success" />
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
        <CodeIcon class="w-5 h-5 text-control" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import {
  CheckCircle2Icon,
  CircleAlertIcon,
  CodeIcon,
  MinusIcon,
  PencilIcon,
  PlayCircleIcon,
  PlayIcon,
  PlusIcon,
  ThumbsUpIcon,
} from "lucide-vue-next";
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import UserAvatar from "@/components/User/UserAvatar.vue";
import {
  extractUserId,
  getIssueCommentType,
  IssueCommentType,
  useUserStore,
} from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { IssueComment_Approval_Status } from "@/types/proto-es/v1/issue_service_pb";

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
  issueComment: IssueComment;
}>();

const userStore = useUserStore();

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier(props.issueComment.creator);
});

const icon = computed((): ActionIconType => {
  const { issueComment } = props;
  const commentType = getIssueCommentType(issueComment);
  if (
    commentType === IssueCommentType.APPROVAL &&
    issueComment.event?.case === "approval"
  ) {
    const { status } = issueComment.event.value;
    switch (status) {
      case IssueComment_Approval_Status.APPROVED:
        return "approve-review";
      case IssueComment_Approval_Status.REJECTED:
        return "reject-review";
      case IssueComment_Approval_Status.PENDING:
        return "re-request-review";
    }
  } else if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    issueComment.event?.case === "issueUpdate"
  ) {
    const { toTitle, toDescription, toLabels, fromLabels } =
      issueComment.event.value;
    if (
      toTitle !== undefined ||
      toDescription !== undefined ||
      toLabels.length !== 0 ||
      fromLabels.length !== 0
    ) {
      return "update";
    }
    // Otherwise, show avatar icon based on the creator.
  } else if (commentType === IssueCommentType.PLAN_SPEC_UPDATE) {
    return "update";
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
