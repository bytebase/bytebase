<template>
  <div class="flex-1 flex w-full">
    <!-- Left Panel - Activity -->
    <div class="flex-1 shrink px-4 py-4 space-y-4">
      <DescriptionSection />

      <OverviewSection v-if="shouldShowOverview" />

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
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContextWithIssue } from "../..";
import { ActivitySection } from "./ActivitySection";
import { DescriptionSection } from "./DescriptionSection";
import OverviewSection from "./OverviewSection.vue";
import { Sidebar } from "./Sidebar";

const { project, ready } = useCurrentProjectV1();
const { plan, issue, rollout } = usePlanContextWithIssue();
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

const shouldShowOverview = computed(() => {
  return issue.value.type === Issue_Type.DATABASE_CHANGE || rollout?.value;
});
// TODO(steven): remove ComposedIssue.
const composedIssue = computed(() => {
  const composedIssue = issue.value as ComposedIssue;
  composedIssue.project = unref(project).name;
  composedIssue.plan = unref(plan).name;
  composedIssue.planEntity = unref(plan);
  if (rollout?.value) {
    composedIssue.rollout = rollout.value.name;
    composedIssue.rolloutEntity = rollout.value;
  }
  return composedIssue;
});

provideIssueContext(
  {
    isCreating: computed(() => false),
    ready,
    allowChange,
    issue: composedIssue,
    ...issueBaseContext,
  },
  true /* root */
);
</script>
