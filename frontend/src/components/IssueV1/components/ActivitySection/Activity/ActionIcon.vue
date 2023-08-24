<template>
  <div class="relative">
    <div v-if="icon == 'system'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <img class="mt-1" src="@/assets/logo-icon.svg" alt="Bytebase" />
      </div>
    </div>
    <div v-else-if="icon == 'avatar'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PrincipalAvatar
          :username="user?.title"
          override-class="w-7 h-7 font-medium"
          override-text-size="0.8rem"
        />
      </div>
    </div>
    <div v-else-if="icon == 'create'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-solid:plus-sm class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon == 'update'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-solid:pencil class="w-4 h-4 text-control" />
      </div>
    </div>
    <div v-else-if="icon == 'run' || icon == 'rollout'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:play class="w-6 h-6 text-control" />
      </div>
    </div>
    <div v-else-if="icon == 'approve-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:thumb-up class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon == 'reject-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-warning rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons:pause-solid class="w-5 h-5 text-white" />
      </div>
    </div>
    <div v-else-if="icon == 're-request-review'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PrincipalAvatar
          :username="user?.title"
          override-class="w-7 h-7 font-medium"
          override-text-size="0.8rem"
        />
      </div>
    </div>
    <div v-else-if="icon == 'cancel'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:minus class="w-5 h-5 text-control" />
      </div>
    </div>
    <div v-else-if="icon == 'fail'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:exclamation-circle class="w-6 h-6 text-error" />
      </div>
    </div>
    <div v-else-if="icon == 'complete'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:check-circle class="w-6 h-6 text-success" />
      </div>
    </div>
    <div v-else-if="icon == 'skip'" class="relative pl-1">
      <div
        class="w-6 h-6 bg-gray-200 rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <SkipIcon class="w-5 h-5 text-gray-500" />
      </div>
    </div>
    <div v-else-if="icon == 'commit'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:code class="w-5 h-5 text-control" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SkipIcon } from "@/components/Icon";
import PrincipalAvatar from "@/components/PrincipalAvatar.vue";
import { useUserStore } from "@/store";
import {
  ActivityIssueCommentCreatePayload,
  ActivityPipelineTaskRunStatusUpdatePayload,
  ActivityStageStatusUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  SYSTEM_BOT_EMAIL,
} from "@/types";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";

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
  activity: LogEntity;
}>();

const user = computed(() => {
  const email = extractUserResourceName(props.activity.creator);
  return useUserStore().getUserByEmail(email);
});

const icon = computed((): ActionIconType => {
  const { activity } = props;
  if (activity.action == LogEntity_Action.ACTION_ISSUE_CREATE) {
    return "create";
  } else if (activity.action == LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE) {
    return "update";
  } else if (
    activity.action == LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE
  ) {
    const payload = JSON.parse(
      activity.payload
    ) as ActivityTaskStatusUpdatePayload;
    switch (payload.newStatus) {
      case "PENDING": {
        if (payload.oldStatus == "RUNNING") {
          return "cancel";
        } else if (payload.oldStatus == "PENDING_APPROVAL") {
          return "rollout";
        }
        break;
      }
      case "CANCELED": {
        return "cancel";
      }
      case "RUNNING": {
        return "run";
      }
      case "DONE": {
        if (payload.oldStatus === "RUNNING") {
          return "complete";
        } else {
          return "skip";
        }
      }
      case "FAILED": {
        return "fail";
      }
      case "SKIPPED": {
        return "skip";
      }
      case "PENDING_APPROVAL": {
        return "avatar"; // stale approval dismissed.
      }
    }
  } else if (
    activity.action === LogEntity_Action.ACTION_PIPELINE_TASK_RUN_STATUS_UPDATE
  ) {
    const payload = JSON.parse(
      activity.payload
    ) as ActivityPipelineTaskRunStatusUpdatePayload;

    switch (payload.newStatus) {
      case "PENDING": {
        return "rollout";
      }
      case "CANCELED": {
        return "cancel";
      }
      case "RUNNING": {
        return "run";
      }
      case "DONE": {
        return "complete";
      }
      case "FAILED": {
        return "fail";
      }
    }
  } else if (
    activity.action == LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE
  ) {
    const payload = JSON.parse(
      activity.payload
    ) as ActivityStageStatusUpdatePayload;
    switch (payload.stageStatusUpdateType) {
      case "BEGIN": {
        return "run";
      }
      case "END": {
        return "complete";
      }
    }
  } else if (
    activity.action == LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT
  ) {
    return "commit";
  } else if (
    activity.action == LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE
  ) {
    return "update";
  } else if (
    activity.action ==
    LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE
  ) {
    return "update";
  } else if (activity.action === LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE) {
    const payload = JSON.parse(
      activity.payload
    ) as ActivityIssueCommentCreatePayload;
    if (payload.approvalEvent) {
      const { status } = payload.approvalEvent;
      switch (status) {
        case "APPROVED":
          return "approve-review";
        case "REJECTED":
          return "reject-review";
        case "PENDING":
          return "re-request-review";
      }
    }
  }

  return extractUserResourceName(activity.creator) == SYSTEM_BOT_EMAIL
    ? "system"
    : "avatar";
});
</script>
