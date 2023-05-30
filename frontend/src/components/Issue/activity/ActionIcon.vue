<template>
  <div class="relative">
    <div v-if="icon == 'system'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <img class="mt-1" src="../../../assets/logo-icon.svg" alt="Bytebase" />
      </div>
    </div>
    <div v-else-if="icon == 'avatar'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-white rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <PrincipalAvatar :principal="activity.creator" :size="'SMALL'" />
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
    <div v-else-if="icon == 'approve'" class="relative pl-0.5">
      <div
        class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
      >
        <heroicons-outline:thumb-up class="w-5 h-5 text-control" />
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
import {
  Activity,
  ActivityIssueCommentCreatePayload,
  ActivityStageStatusUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  SYSTEM_BOT_ID,
} from "@/types";
import { SkipIcon } from "@/components/Icon";

type ActionIconType =
  | "avatar"
  | "system"
  | "create"
  | "update"
  | "run"
  | "approve"
  | "rollout"
  | "cancel"
  | "fail"
  | "complete"
  | "skip"
  | "commit";

const props = defineProps<{
  activity: Activity;
}>();

const icon = computed((): ActionIconType => {
  const { activity } = props;
  if (activity.type == "bb.issue.create") {
    return "create";
  } else if (activity.type == "bb.issue.field.update") {
    return "update";
  } else if (activity.type == "bb.pipeline.task.status.update") {
    const payload = activity.payload as ActivityTaskStatusUpdatePayload;
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
      case "PENDING_APPROVAL": {
        return "avatar"; // stale approval dismissed.
      }
    }
  } else if (activity.type == "bb.pipeline.stage.status.update") {
    const payload = activity.payload as ActivityStageStatusUpdatePayload;
    switch (payload.stageStatusUpdateType) {
      case "BEGIN": {
        return "run";
      }
      case "END": {
        return "complete";
      }
    }
  } else if (activity.type == "bb.pipeline.task.file.commit") {
    return "commit";
  } else if (activity.type == "bb.pipeline.task.statement.update") {
    return "update";
  } else if (
    activity.type == "bb.pipeline.task.general.earliest-allowed-time.update"
  ) {
    return "update";
  } else if (activity.type === "bb.issue.comment.create") {
    const payload = activity.payload as ActivityIssueCommentCreatePayload;
    if (payload.approvalEvent?.status === "APPROVED") {
      return "approve";
    }
  }

  return activity.creator.id == SYSTEM_BOT_ID ? "system" : "avatar";
});
</script>
