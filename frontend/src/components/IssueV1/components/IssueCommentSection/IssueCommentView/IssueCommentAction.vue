<template>
  <div class="ml-3 min-w-0 flex-1">
    <div class="rounded-lg border border-gray-200 bg-white overflow-hidden">
      <div class="px-4 py-2.5 bg-gray-50">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-x-2 text-sm min-w-0 flex-wrap">
            <ActionCreator
              v-if="
                extractUserId(issueComment.creator) !==
                  userStore.systemBotUser?.email ||
                getIssueCommentType(issueComment) ===
                  IssueCommentType.USER_COMMENT
              "
              :creator="issueComment.creator"
            />

            <ActionSentence
              :issue="issue"
              :issue-comment="issueComment"
              class="text-gray-600 break-words min-w-0"
            />

            <HumanizeTs
              :ts="
                getTimeForPbTimestampProtoEs(issueComment.createTime, 0) / 1000
              "
              class="text-gray-500"
            />

            <span
              v-if="
                getTimeForPbTimestampProtoEs(issueComment.createTime) !==
                  getTimeForPbTimestampProtoEs(issueComment.updateTime) &&
                getIssueCommentType(issueComment) ===
                  IssueCommentType.USER_COMMENT
              "
              class="text-gray-500 text-xs"
            >
              ({{ $t("common.edited") }})
            </span>

            <span v-if="similar.length > 0" class="text-xs text-gray-500">
              {{
                $t("activity.n-similar-activities", {
                  count: similar.length + 1,
                })
              }}
            </span>
          </div>

          <slot name="subject-suffix"></slot>
        </div>
      </div>
      <div
        v-if="$slots.comment"
        class="px-4 py-3 border-t border-gray-200 text-sm text-gray-700 whitespace-pre-wrap break-words"
      >
        <slot name="comment" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import { IssueCommentType, getIssueCommentType } from "@/store";
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
