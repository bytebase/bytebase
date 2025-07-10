<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h3 class="text-base font-medium">
        {{ $t("common.activity") }}
      </h3>
    </div>
    <IssueCommentList :comment-filter="commentFilter" />
  </div>
</template>

<script setup lang="ts">
import { includes } from "lodash-es";
import IssueCommentList from "@/components/IssueV1/components/IssueCommentSection/IssueCommentList.vue";
import { IssueCommentType, type ComposedIssueComment } from "@/store";

// Only show issue related comments.
const commentFilter = (comment: ComposedIssueComment): boolean => {
  return includes(
    [
      IssueCommentType.USER_COMMENT,
      IssueCommentType.APPROVAL,
      IssueCommentType.ISSUE_UPDATE,
    ],
    comment.type
  );
};
</script>
