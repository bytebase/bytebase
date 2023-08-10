<template>
  <div class="flex-1">
    <NInput
      :value="state.title"
      :style="style"
      :loading="state.isUpdating"
      :disabled="!allowEdit || state.isUpdating"
      size="large"
      required
      class="bb-issue-title-input"
      @focus="state.isEditing = true"
      @blur="onBlur"
      @keyup.enter="onEnter"
      @update:value="onUpdateValue"
    />
  </div>
</template>

<script setup lang="ts">
import { NInput } from "naive-ui";
import { CSSProperties, computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { Issue, IssueStatus } from "@/types/proto/v1/issue_service";
import { extractUserResourceName, hasWorkspacePermissionV1 } from "@/utils";
import { useIssueContext } from "../../logic";

type ViewMode = "EDIT" | "VIEW";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { isCreating, issue } = useIssueContext();

const state = reactive({
  isEditing: false,
  isUpdating: false,
  title: issue.value.title,
});

const viewMode = computed((): ViewMode => {
  if (isCreating.value) return "EDIT";
  return state.isEditing ? "EDIT" : "VIEW";
});

const style = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color-disabled": "transparent",
    "--n-font-size": "18px",
    "font-weight": "bold",
  };
  const border =
    viewMode.value === "EDIT"
      ? "1px solid var(--color-control-border)"
      : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const allowEdit = computed(() => {
  if (isCreating.value) {
    return true;
  }

  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  if (
    extractUserResourceName(issue.value.assignee) === currentUser.value.email ||
    extractUserResourceName(issue.value.creator) === currentUser.value.email
  ) {
    // Allowed if current user is the assignee or creator.
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      currentUser.value.userRole
    )
  ) {
    // Allowed if RBAC is enabled and current is DBA or workspace owner.
    return true;
  }
  return false;
});

const onBlur = async () => {
  const cleanup = () => {
    state.isEditing = false;
    state.isUpdating = false;
  };

  if (isCreating.value) {
    cleanup();
    return;
  }
  if (state.title === issue.value.title) {
    cleanup();
    return;
  }
  try {
    state.isUpdating = true;
    // TODO update name
    const issuePatch = Issue.fromJSON({
      ...issue.value,
      title: state.title,
    });
    const updated = await issueServiceClient.updateIssue({
      issue: issuePatch,
      updateMask: ["title"],
    });
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    cleanup();
  }
};

const onEnter = (e: Event) => {
  const input = e.target as HTMLInputElement;
  input.blur();
};

const onUpdateValue = (title: string) => {
  state.title = title;
  if (isCreating.value) {
    issue.value.title = title;
  }
};

watch(
  () => issue.value.title,
  (title) => {
    state.title = title;
  }
);
</script>

<style>
.bb-issue-title-input input {
  cursor: text !important;
  color: var(--n-text-color) !important;
  text-decoration-color: var(--n-text-color) !important;
}
</style>
