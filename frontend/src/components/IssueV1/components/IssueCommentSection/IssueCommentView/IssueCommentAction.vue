<template>
  <div class="ml-3 min-w-0 flex-1">
    <div class="min-w-0 flex-1 pt-1 flex justify-between">
      <div class="text-sm text-control-light space-x-1">
        <ActionCreator
          v-if="
            extractUserId(issueComment.creator) !==
              userStore.systemBotUser?.email ||
            issueComment.type === IssueCommentType.USER_COMMENT
          "
          :creator="issueComment.creator"
        />

        <ActionSentence :issue="issue" :issue-comment="issueComment" />

        <HumanizeTs
          :ts="getTimeForPbTimestamp(issueComment.createTime, 0) / 1000"
          class="ml-1 text-gray-400"
        />

        <span
          v-if="
            getTimeForPbTimestamp(issueComment.createTime) !==
              getTimeForPbTimestamp(issueComment.updateTime) &&
            issueComment.type === IssueCommentType.USER_COMMENT
          "
        >
          <span>({{ $t("common.edited") }}</span>
          <HumanizeTs
            :ts="getTimeForPbTimestamp(issueComment.updateTime, 0) / 1000"
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
import { IssueCommentType, type ComposedIssueComment } from "@/store";
import { useUserStore, extractUserId } from "@/store";
import { getTimeForPbTimestamp, type ComposedIssue } from "@/types";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";

defineProps<{
  issue: ComposedIssue;
  index: number;
  issueComment: ComposedIssueComment;
  similar: ComposedIssueComment[];
}>();

const userStore = useUserStore();
</script>
