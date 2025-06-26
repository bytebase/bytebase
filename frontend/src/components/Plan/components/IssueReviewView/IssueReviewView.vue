<template>
  <div class="flex-1 flex w-full">
    <!-- Left Panel - Activity -->
    <div class="flex-1 shrink">
      <ActivitySection />
    </div>

    <div class="w-80 shrink-0 flex flex-col border-l">
      <Sidebar />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, unref } from "vue";
import { provideIssueContext, useBaseIssueContext } from "@/components/IssueV1";
import { extractUserId, useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContextWithIssue } from "../..";
import { ActivitySection } from "./ActivitySection";
import { Sidebar } from "./Sidebar";

const { project, ready } = useCurrentProjectV1();
const { issue } = usePlanContextWithIssue();
const currentUser = useCurrentUserV1();

const issueBaseContext = useBaseIssueContext({
  // Always set to false.
  isCreating: computed(() => false),
  ready,
  issue: computed(() => issue.value as ComposedIssue),
});

const allowChange = computed(() => {
  // Disallow changes if the issue is not open.
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  // Allow changes if the current user is the creator of the issue or has the necessary permissions.
  return (
    extractUserId(issue.value.creator) === currentUser.value.email ||
    hasProjectPermissionV2(unref(project), "bb.issues.update")
  );
});

provideIssueContext(
  {
    isCreating: computed(() => false),
    ready,
    allowChange,
    issue: computed(() => issue.value as ComposedIssue),
    ...issueBaseContext,
  },
  true /* root */
);
</script>
