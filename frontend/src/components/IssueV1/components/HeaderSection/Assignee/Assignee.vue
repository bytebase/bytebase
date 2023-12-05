<template>
  <div class="flex items-center justify-end gap-3">
    <div class="flex items-center justify-end gap-1">
      <NTooltip>
        <template #trigger>
          <div class="flex items-center gap-x-1 textlabel">
            <span>{{ $t("common.assignee") }}</span>
            <span v-if="isCreating" class="text-red-600">*</span>
          </div>
        </template>
        <template #default>
          <div class="max-w-[14rem]">
            {{ $t("issue.assignee-tooltip") }}
          </div>
        </template>
      </NTooltip>

      <!-- AssigneeAttentionButton will be now shown until the feature is re-defined -->
      <AssigneeAttentionButton v-if="false" />
    </div>

    <NTooltip :disabled="errors.length === 0">
      <template #trigger>
        <UserSelect
          :multiple="false"
          :user="assigneeUID"
          :disabled="errors.length > 0 || isUpdating"
          :loading="isUpdating"
          :filter="filterAssignee"
          :include-system-bot="false"
          :map-options="mapUserOptions"
          :fallback-option="fallbackUser"
          :clearable="true"
          :auto-reset="false"
          style="width: 14rem"
          @update:user="changeAssigneeUID"
        />
      </template>
      <template #default>
        <ErrorList :errors="errors" class="max-w-[24rem] !whitespace-normal" />
      </template>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { NTooltip, SelectGroupOption } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  allowUserToChangeAssignee,
  useIssueContext,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1/logic";
import ErrorList, { ErrorItem } from "@/components/misc/ErrorList.vue";
import { UserSelect } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import {
  SYSTEM_BOT_EMAIL,
  SYSTEM_BOT_ID,
  SYSTEM_BOT_USER_NAME,
  UNKNOWN_ID,
  unknownUser,
} from "@/types";
import { User, UserRole } from "@/types/proto/v1/auth_service";
import { Issue } from "@/types/proto/v1/issue_service";
import {
  extractUserResourceName,
  extractUserUID,
  hasWorkspacePermissionV1,
  isMemberOfProjectV1,
  isOwnerOfProjectV1,
} from "@/utils";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const { t } = useI18n();
const userStore = useUserStore();
const { isCreating, issue, reviewContext, releaserCandidates } =
  useIssueContext();
const currentUser = useCurrentUserV1();
const isUpdating = ref(false);

const assigneeEmail = computed(() => {
  const assignee = issue.value.assignee;
  if (!assignee) return undefined;
  return extractUserResourceName(assignee);
});

const assigneeUID = computed(() => {
  if (!assigneeEmail.value) return undefined;
  if (assigneeEmail.value === SYSTEM_BOT_EMAIL) return String(SYSTEM_BOT_ID);
  const user = userStore.getUserByEmail(assigneeEmail.value);
  if (!user) return undefined;
  return extractUserUID(user.name);
});

const allowChangeAssignee = computed(() => {
  if (isCreating.value) {
    return true;
  }
  return allowUserToChangeAssignee(currentUser.value, issue.value);
});

const errors = asyncComputed(async () => {
  const errors: ErrorItem[] = [];
  if (!allowChangeAssignee.value) {
    errors.push(t("issue.you-are-not-allowed-to-change-this-value"));
  }
  return errors;
}, []);

const changeAssigneeUID = async (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    issue.value.assignee = "";
  } else {
    const assignee = userStore.getUserById(uid);
    if (!assignee) {
      issue.value.assignee = "";
    } else {
      issue.value.assignee = `users/${assignee.email}`;
    }
  }

  if (!isCreating.value) {
    const issuePatch = Issue.fromJSON(issue.value);
    isUpdating.value = true;
    try {
      const updated = await issueServiceClient.updateIssue({
        issue: issuePatch,
        updateMask: ["assignee"],
      });
      Object.assign(issue.value, updated);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      emitWindowEvent("bb.issue-field-update");
    } finally {
      isUpdating.value = false;
    }
  }
};

const filterAssignee = (user: User): boolean => {
  return (
    isMemberOfProjectV1(issue.value.projectEntity.iamPolicy, user) ||
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      user.userRole
    )
  );
};

const mapUserOptions = (users: User[]) => {
  const project = issue.value.projectEntity;
  const phase = isCreating.value
    ? "PREVIEW"
    : reviewContext.done.value
    ? "CD"
    : "CI";
  const added = new Set<string>(); // by user.name
  const groups: SelectGroupOption[] = [];
  const mapUserOption = (user: User) => ({
    user,
    value: extractUserUID(user.name),
    label: user.title,
  });
  // `addGroup` ensures that
  // 1. any users will be added at most once
  // 2. empty groups will not be added
  const addGroup = (group: SelectGroupOption, users: User[]) => {
    const filteredUsers = users.filter((u) => !added.has(u.name));
    if (filteredUsers.length > 0) {
      filteredUsers.forEach((u) => added.add(u.name));
      group.children = filteredUsers.map(mapUserOption);
      groups.push(group);
    }
  };

  // Groups in order
  // - current assignee
  // - approvers of current step (CI phase only)
  // - releasers of current stage (CD phase only)
  // - project owners
  // - other non-member assignee candidates
  //   - workspace owners
  //   - workspace DBAs
  if (assigneeEmail.value) {
    const assignee = userStore.getUserByEmail(assigneeEmail.value);
    if (assignee) {
      addGroup(
        {
          type: "group",
          label: t("issue.assignee.current-assignee"),
          key: "current-assignee",
        },
        [assignee]
      );
    }
  }

  if (phase === "CI") {
    const steps = useWrappedReviewStepsV1(issue, reviewContext);
    const currentStep = steps.value?.find((step) => step.status === "CURRENT");
    addGroup(
      {
        type: "group",
        label: t("issue.assignee.approvers-of-current-step"),
        key: "approvers",
      },
      currentStep?.candidates ?? []
    );
  }
  if (phase === "CD") {
    const candidates = new Set(releaserCandidates.value.map((c) => c.name));
    const releasers = users.filter((user) => candidates.has(user.name));
    addGroup(
      {
        type: "group",
        label: t("issue.assignee.releasers-of-current-stage"),
        key: "releasers",
      },
      releasers
    );
  }
  const projectOwners = users.filter((user) =>
    isOwnerOfProjectV1(project.iamPolicy, user)
  );
  addGroup(
    {
      type: "group",
      label: t("issue.assignee.project-owners"),
      key: "project-owners",
    },
    projectOwners
  );

  // Add non-project members (workspace admins and DBAs)
  const workspaceOwners = users.filter(
    (user) => user.userRole === UserRole.OWNER
  );
  addGroup(
    {
      type: "group",
      label: t("issue.assignee.workspace-admins"),
      key: "workspace-admins",
    },
    workspaceOwners
  );
  const workspaceDBAs = users.filter(
    (user) =>
      !isMemberOfProjectV1(project.iamPolicy, user) &&
      user.userRole === UserRole.DBA
  );
  addGroup(
    {
      type: "group",
      label: t("issue.assignee.workspace-dbas"),
      key: "workspace-dbas",
    },
    workspaceDBAs
  );

  return groups;
};

const fallbackUser = (uid: string) => {
  if (uid === String(SYSTEM_BOT_ID)) {
    return {
      user: userStore.getUserByName(SYSTEM_BOT_USER_NAME)!,
      value: uid,
    };
  }
  return {
    user: unknownUser(),
    value: uid,
  };
};
</script>
