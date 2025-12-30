<template>
  <div class="relative">
    <div v-if="iconConfig.type === 'system'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <img class="mt-1" src="@/assets/logo-icon.svg" alt="Bytebase" />
      </div>
    </div>
    <div v-else-if="iconConfig.type === 'avatar'" class="relative pl-0.5">
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
    <div v-else class="relative" :class="iconConfig.wrapper">
      <div
        class="rounded-full ring-4 ring-white flex items-center justify-center"
        :class="iconConfig.container"
      >
        <component :is="iconConfig.icon" :class="iconConfig.iconClass" />
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
import type { Component } from "vue";
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

interface IconConfig {
  type: "system" | "avatar" | "icon";
  wrapper?: string;
  container?: string;
  icon?: Component;
  iconClass?: string;
}

const ICON_CONFIGS: Record<
  Exclude<ActionIconType, "system" | "avatar">,
  Omit<IconConfig, "type">
> = {
  create: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: PlusIcon,
    iconClass: "w-5 h-5 text-control",
  },
  update: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: PencilIcon,
    iconClass: "w-4 h-4 text-control",
  },
  run: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: PlayCircleIcon,
    iconClass: "w-5 h-5 text-control",
  },
  rollout: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: PlayCircleIcon,
    iconClass: "w-5 h-5 text-control",
  },
  "approve-review": {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-success",
    icon: ThumbsUpIcon,
    iconClass: "w-4 h-4 text-white",
  },
  "reject-review": {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-warning",
    icon: PencilIcon,
    iconClass: "w-4 h-4 text-white",
  },
  "re-request-review": {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg icon-re-request-review",
    icon: PlayIcon,
    iconClass: "w-4 h-4 text-control ml-px",
  },
  cancel: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: MinusIcon,
    iconClass: "w-5 h-5 text-control",
  },
  fail: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-white",
    icon: CircleAlertIcon,
    iconClass: "w-5 h-5 text-error",
  },
  complete: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-white",
    icon: CheckCircle2Icon,
    iconClass: "w-6 h-6 text-success",
  },
  skip: {
    wrapper: "pl-1",
    container: "w-6 h-6 bg-gray-200",
    icon: SkipIcon,
    iconClass: "w-5 h-5 text-gray-500",
  },
  commit: {
    wrapper: "pl-0.5",
    container: "w-7 h-7 bg-control-bg",
    icon: CodeIcon,
    iconClass: "w-5 h-5 text-control",
  },
};

const props = defineProps<{
  issueComment: IssueComment;
}>();

const userStore = useUserStore();

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier(props.issueComment.creator);
});

const iconType = computed((): ActionIconType => {
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
  }

  if (
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
  }

  if (commentType === IssueCommentType.PLAN_SPEC_UPDATE) {
    return "update";
  }

  return extractUserId(issueComment.creator) === userStore.systemBotUser?.email
    ? "system"
    : "avatar";
});

const iconConfig = computed((): IconConfig => {
  const type = iconType.value;
  if (type === "system" || type === "avatar") {
    return { type };
  }
  return { type: "icon", ...ICON_CONFIGS[type] };
});
</script>

<style scoped>
.icon-re-request-review :deep(path) {
  stroke-width: 3 !important;
}
</style>
