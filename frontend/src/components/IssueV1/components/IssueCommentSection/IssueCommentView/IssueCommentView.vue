<template>
  <li>
    <div :id="`#${issueComment.name}`" class="relative pb-4">
      <span
        v-if="!isLast"
        class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
        aria-hidden="true"
      ></span>
      <div class="relative flex items-start">
        <ActionIcon :issue-comment="issueComment" />

        <IssueCommentAction
          :issue="issue"
          :index="index"
          :issue-comment="issueComment"
          :similar="similar"
        >
          <template #subject-suffix>
            <slot name="subject-suffix" />
          </template>

          <template #comment>
            <slot name="comment" />
          </template>
        </IssueCommentAction>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import type { ComposedIssueComment } from "@/store";
import type { ComposedIssue } from "@/types";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import ActionIcon from "./ActionIcon.vue";
import IssueCommentAction from "./IssueCommentAction.vue";

defineProps<{
  issue: ComposedIssue;
  isLast: boolean;
  index: number;
  issueComment: ComposedIssueComment;
  similar: ComposedIssueComment[];
  rollout?: Rollout;
}>();
</script>
