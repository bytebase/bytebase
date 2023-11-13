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
import { orderBy } from "lodash-es";
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
  isMemberOfProjectV1,
  isOwnerOfProjectV1,
} from "@/utils";
import AssigneeAttentionButton from "./AssigneeAttentionButton.vue";

const { t } = useI18n();
const userStore = useUserStore();
const { isCreating, issue, reviewContext, assigneeCandidates } =
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
    return;
  }
  const assignee = userStore.getUserById(uid);
  if (!assignee) {
    issue.value.assignee = "";
    return;
  }
  issue.value.assignee = `users/${assignee.email}`;

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
  return isMemberOfProjectV1(issue.value.projectEntity.iamPolicy, user);
};

const mapUserOptions = (users: User[]) => {
  const project = issue.value.projectEntity;

  // Project owners go top
  users = orderBy(
    users,
    [(user) => (isOwnerOfProjectV1(project.iamPolicy, user) ? -1 : 1)],
    ["asc"]
  );

  const phase = isCreating.value
    ? "PREVIEW"
    : reviewContext.done.value
    ? "CD"
    : "CI";
  const groups: SelectGroupOption[] = [];

  const mapUserOption = (user: User) => ({
    user,
    value: extractUserUID(user.name),
    label: user.title,
  });

  const members = users.filter((user) =>
    isMemberOfProjectV1(project.iamPolicy, user)
  );

  if (phase === "PREVIEW") {
    const owners = members.filter((user) =>
      isOwnerOfProjectV1(project.iamPolicy, user)
    );
    if (owners.length > 0) {
      groups.push({
        type: "group",
        label: t("issue.assignee.project-owners"),
        key: "project-owners",
        children: owners.map(mapUserOption),
      });
    }
    const nonOwnerMembers = members.filter(
      (user) => !isOwnerOfProjectV1(project.iamPolicy, user)
    );
    if (owners.length > 0) {
      groups.push({
        type: "group",
        label: t("issue.assignee.project-members"),
        key: "project-members",
        children: nonOwnerMembers.map(mapUserOption),
      });
    }
  }

  if (phase === "CI") {
    const steps = useWrappedReviewStepsV1(issue, reviewContext);
    const currentStep = steps.value?.find((step) => step.status === "CURRENT");
    if (currentStep && currentStep.candidates.length > 0) {
      const approverGroup: SelectGroupOption = {
        type: "group",
        label: t("issue.assignee.approvers"),
        key: "approvers",
        children: currentStep.candidates.map(mapUserOption),
      };
      groups.push(approverGroup);
    }
    const nonApprovers = members.filter((user) => {
      if (!currentStep) return true;
      return currentStep.candidates.findIndex((c) => c.name === user.name) < 0;
    });
    if (nonApprovers.length > 0) {
      const nonApproverGroup: SelectGroupOption = {
        type: "group",
        label: t("issue.assignee.project-members"),
        key: "project-members",
        children: nonApprovers.map(mapUserOption),
      };
      groups.push(nonApproverGroup);
    }
  }

  if (phase === "CD") {
    const releasers = members.filter((user) => {
      return (
        assigneeCandidates.value.findIndex((c) => c.name === user.name) >= 0
      );
    });
    const nonReleasers = members.filter(
      (user) => releasers.findIndex((r) => r.name === user.name) < 0
    );
    if (releasers.length > 0) {
      const releaserGroup: SelectGroupOption = {
        type: "group",
        label: t("issue.assignee.releaser-of-current-stage"),
        key: "releasers",
        children: releasers.map(mapUserOption),
      };
      groups.push(releaserGroup);
    }
    if (nonReleasers.length > 0) {
      const memberGroup: SelectGroupOption = {
        type: "group",
        label: t("issue.assignee.project-members"),
        key: "members",
        children: nonReleasers.map(mapUserOption),
      };
      groups.push(memberGroup);
    }
  }

  const nonMemberWorkspaceOwners = users.filter(
    (user) =>
      !isMemberOfProjectV1(project.iamPolicy, user) &&
      user.userRole === UserRole.OWNER
  );
  if (nonMemberWorkspaceOwners.length > 0) {
    groups.push({
      type: "group",
      label: t("issue.assignee.workspace-owners"),
      key: "workspace-owners",
      children: nonMemberWorkspaceOwners.map(mapUserOption),
    });
  }
  const nonMemberWorkspaceDBAs = users.filter(
    (user) =>
      !isMemberOfProjectV1(project.iamPolicy, user) &&
      user.userRole === UserRole.DBA
  );
  if (nonMemberWorkspaceDBAs.length > 0) {
    groups.push({
      type: "group",
      label: t("issue.assignee.workspace-dbas"),
      key: "workspace-dbas",
      children: nonMemberWorkspaceDBAs.map(mapUserOption),
    });
  }

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
