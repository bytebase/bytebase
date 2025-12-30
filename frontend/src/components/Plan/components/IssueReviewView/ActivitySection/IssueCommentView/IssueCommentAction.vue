<template>
  <div class="min-w-0 flex-1">
    <div
      class="rounded-lg border overflow-hidden"
      :class="
        $slots.comment
          ? 'ml-3 border-gray-200 bg-white'
          : 'ml-1 border-transparent'
      "
    >
      <div class="px-3 py-2" :class="$slots.comment ? 'bg-gray-50' : ''">
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
              :issue-comment="issueComment"
              class="text-gray-600 wrap-break-word min-w-0"
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
          </div>

          <slot name="subject-suffix"></slot>
        </div>
      </div>
      <div
        v-if="$slots.comment"
        class="px-4 py-3 border-t border-gray-200 text-sm text-gray-700 whitespace-pre-wrap wrap-break-word"
      >
        <slot name="comment" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import {
  extractUserId,
  getIssueCommentType,
  IssueCommentType,
  useUserStore,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";

defineProps<{
  issueComment: IssueComment;
}>();

const userStore = useUserStore();
</script>
