<template>
  <div class="px-4 py-2 flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ title }}
      </span>

      <div v-if="!isCreating && allowEdit" class="flex items-center gap-x-2">
        <NButton v-if="!state.isEditing" size="tiny" @click.prevent="beginEdit">
          {{ $t("common.edit") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          :disabled="state.description === issue.description"
          :loading="state.isUpdating"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          quaternary
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </NButton>
      </div>
    </div>

    <div class="text-sm">
      <NInput
        v-if="isCreating || state.isEditing"
        ref="inputRef"
        v-model:value="state.description"
        style="
          width: 100%;
          --n-placeholder-color: var(--color-control-placeholder);
        "
        type="textarea"
        size="small"
        :placeholder="$t('issue.add-some-description')"
        :autosize="{ minRows: 3, maxRows: 10 }"
        :disabled="state.isUpdating"
        @update:value="onDescriptionChange"
      />
      <div
        v-else
        class="min-h-[3rem] max-h-[12rem] whitespace-pre-wrap px-[10px] py-[4.5px] text-sm"
      >
        <template v-if="issue.description">
          {{ issue.description }}
        </template>
        <span v-else class="text-control-placeholder">
          {{ $t("issue.add-some-description") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NInput, NButton } from "naive-ui";

import { useIssueContext } from "../../logic";
import { hasWorkspacePermissionV1, isGrantRequestIssue } from "@/utils";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { extractUserEmail, pushNotification, useCurrentUserV1 } from "@/store";
import { issueServiceClient } from "@/grpcweb";

type LocalState = {
  isEditing: boolean;
  isUpdating: boolean;
  description: string;
};

const { t } = useI18n();
const { isCreating, issue } = useIssueContext();
const currentUser = useCurrentUserV1();

const state = reactive<LocalState>({
  isEditing: false,
  isUpdating: false,
  description: issue.value.description,
});

const inputRef = ref<InstanceType<typeof NInput>>();

const title = computed(() => {
  return isGrantRequestIssue(issue.value)
    ? t("common.reason")
    : t("common.description");
});

const allowEdit = computed(() => {
  if (isCreating.value) {
    return true;
  }
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  if (
    extractUserEmail(issue.value.assignee) === currentUser.value.email ||
    extractUserEmail(issue.value.creator) === currentUser.value.email
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

const onDescriptionChange = (description: string) => {
  if (isCreating.value) {
    issue.value.description = description;
  }
};

const beginEdit = () => {
  state.description = issue.value.description;
  state.isEditing = true;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const saveEdit = async () => {
  try {
    state.isUpdating = true;
    const issuePatch = { ...issue, description: state.description };
    await issueServiceClient.updateIssue({
      issue: issuePatch,
      updateMask: ["description"],
    });
    issue.value.description = state.description;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    state.isEditing = false;
  } finally {
    state.isUpdating = false;
  }
};

const cancelEdit = () => {
  state.description = issue.value.description;
  state.isEditing = false;
};

// Reset the edit state after creating the issue.
watch(isCreating, (curr, prev) => {
  if (!curr && prev) {
    state.isEditing = false;
  }
});

watch(
  () => issue.value,
  (issue) => {
    state.description = issue.description;
  },
  { immediate: true }
);
</script>
