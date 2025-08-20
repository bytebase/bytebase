<template>
  <div class="ml-3 min-w-0 flex-1">
    <h3
      class="sr-only"
      :id="`${issueCommentNamePrefix}${getProjectIdIssueIdIssueCommentId(issueComment.name).issueCommentId}`"
    ></h3>
    <div class="min-w-0 flex-1 pt-1 flex justify-between">
      <div class="text-sm text-control-light space-x-1">
        <ActionCreator
          v-if="
            extractUserId(issueComment.creator) !==
              userStore.systemBotUser?.email ||
            getIssueCommentType(issueComment) === IssueCommentType.USER_COMMENT
          "
          :creator="issueComment.creator"
        />

        <ActionSentence :issue="issue" :issue-comment="issueComment" />

        <HumanizeTs
          :ts="getTimeForPbTimestampProtoEs(issueComment.createTime, 0) / 1000"
          class="ml-1 text-gray-400"
        />

        <span
          v-if="
            getTimeForPbTimestampProtoEs(issueComment.createTime) !==
              getTimeForPbTimestampProtoEs(issueComment.updateTime) &&
            getIssueCommentType(issueComment) === IssueCommentType.USER_COMMENT
          "
        >
          <span class="opacity-80">({{ $t("common.edited") }}</span>
          <HumanizeTs
            :ts="
              getTimeForPbTimestampProtoEs(issueComment.updateTime, 0) / 1000
            "
            class="ml-1"
          />)
        </span>

        <span
          v-if="similar.length > 0"
          class="text-sm font-normal text-gray-400 ml-1"
        >
          {{
            $t("activity.n-similar-activities", {
              count: similar.length + 1,
            })
          }}
        </span>
      </div>

      <slot name="subject-suffix"></slot>
    </div>
    <div class="mt-2 text-sm text-control whitespace-pre-wrap">
      <slot name="comment" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import {
  getProjectIdIssueIdIssueCommentId,
  issueCommentNamePrefix,
  IssueCommentType,
  getIssueCommentType,
} from "@/store";
import { useUserStore, extractUserId } from "@/store";
import { getTimeForPbTimestampProtoEs, type ComposedIssue } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";

defineProps<{
  issue: ComposedIssue;
  index: number;
  issueComment: IssueComment;
  similar: IssueComment[];
  rollout?: Rollout;
}>();

const userStore = useUserStore();
</script>
