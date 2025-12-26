<template>
  <div class="flex-1 flex w-full border-t -mt-px">
    <!-- Left Panel - Activity -->
    <div class="flex-1 shrink p-4 flex flex-col gap-y-4 overflow-x-auto">
      <slot />
      <ActivitySection />
    </div>

    <!-- Desktop Sidebar -->
    <div
      v-if="sidebarMode === 'DESKTOP'"
      class="shrink-0 flex flex-col border-l"
      :style="{
        width: `${desktopSidebarWidth}px`,
      }"
    >
      <Sidebar />
    </div>

    <!-- Mobile Sidebar -->
    <template v-if="sidebarMode === 'MOBILE'">
      <Drawer :show="mobileSidebarOpen" @close="mobileSidebarOpen = false">
        <div
          style="
            min-width: 240px;
            width: 80vw;
            max-width: 320px;
            padding: 0.5rem;
          "
        >
          <Sidebar />
        </div>
      </Drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, unref } from "vue";
import { provideIssueContext, useBaseIssueContext } from "@/components/IssueV1";
import { Drawer } from "@/components/v2";
import { extractUserId, useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContextWithIssue } from "../..";
import { useSidebarContext } from "../../logic/sidebar";
import { ActivitySection } from "./ActivitySection";
import { Sidebar } from "./Sidebar";

const { project, ready } = useCurrentProjectV1();
const { plan, issue, rollout } = usePlanContextWithIssue();
const currentUser = useCurrentUserV1();

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = useSidebarContext();

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

// TODO(steven): remove ComposedIssue.
const composedIssue = computed(() => {
  const composedIssue = issue.value as ComposedIssue;
  composedIssue.project = unref(project).name;
  composedIssue.plan = unref(plan).name;
  composedIssue.planEntity = unref(plan);
  if (rollout?.value) {
    composedIssue.rolloutEntity = rollout.value;
  }
  return composedIssue;
});

provideIssueContext({
  isCreating: computed(() => false),
  ready,
  allowChange,
  issue: composedIssue,
  ...issueBaseContext,
});
</script>
