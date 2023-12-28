<template>
  <NTooltip :disabled="errors.length === 0">
    <template #trigger>
      <NSwitch
        :value="checked"
        :disabled="!allowChange"
        :loading="isUpdating"
        class="bb-ghost-switch"
        @update:value="toggleChecked"
      >
        <template #checked>
          <span style="font-size: 10px">{{ $t("common.on") }}</span>
        </template>
        <template #unchecked>
          <span style="font-size: 10px">{{ $t("common.off") }}</span>
        </template>
      </NSwitch>
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>

  <InstanceAssignment
    v-if="showMissingInstanceLicense"
    :show="showInstanceAssignmentDrawer"
    @dismiss="showInstanceAssignmentDrawer = false"
  />
</template>

<script setup lang="ts">
import { NSwitch, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { specForTask, useIssueContext } from "@/components/IssueV1/logic";
import {
  ErrorItem,
  default as ErrorList,
} from "@/components/misc/ErrorList.vue";
import { hasFeature, useCurrentUserV1 } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  MIN_GHOST_SUPPORT_MYSQL_VERSION,
  engineNameV1,
  flattenTaskV1List,
  hasWorkspacePermissionV1,
} from "@/utils";
import { allowGhostForTask, useIssueGhostContext } from "./common";

const { t } = useI18n();
const me = useCurrentUserV1();
const { isCreating, issue, selectedTask: task } = useIssueContext();
const { viewType, toggleGhost, showFeatureModal, showMissingInstanceLicense } =
  useIssueGhostContext();
const isUpdating = ref(false);
const showInstanceAssignmentDrawer = ref(false);

const allowChange = computed(() => {
  return isCreating.value;
});

const checked = computed(() => {
  return viewType.value === "ON";
});

const canManageSubscription = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-subscription",
    me.value.userRole
  );
});

const allowGhostForEveryDatabase = computed(() => {
  const tasks = flattenTaskV1List(issue.value.rolloutEntity);
  return tasks.every((task) => allowGhostForTask(issue.value, task));
});

const errors = computed(() => {
  const errors: ErrorItem[] = [];
  if (showMissingInstanceLicense.value && !canManageSubscription.value) {
    // Only show the tooltip when current user is not allowed to manage subscription
    // since we will show the InstanceAssignmentDrawer for high-privileged users
    // when clicking on the switch
    errors.push(
      t("subscription.instance-assignment.missing-license-attention")
    );
  }
  if (!allowGhostForEveryDatabase.value) {
    errors.push(
      t(
        "task.online-migration.error.not-applicable.some-tasks-dont-meet-ghost-requirement"
      )
    );
    errors.push({
      error: `${engineNameV1(
        Engine.MYSQL
      )} >= ${MIN_GHOST_SUPPORT_MYSQL_VERSION}`,
      indent: 1,
    });
  }
  return errors;
});

const toggleChecked = async (on: boolean) => {
  if (!hasFeature("bb.feature.online-migration")) {
    showFeatureModal.value = true;
    return;
  }
  if (showMissingInstanceLicense.value) {
    if (canManageSubscription.value) {
      showInstanceAssignmentDrawer.value = true;
    }
    return;
  }
  if (errors.value.length > 0) {
    return;
  }

  const spec = specForTask(issue.value.planEntity, task.value);
  if (!spec) return;
  isUpdating.value = true;
  try {
    await toggleGhost(spec, on);
  } finally {
    isUpdating.value = false;
  }
};
</script>

<style lang="postcss" scoped>
.bb-ghost-switch {
  --n-width: max(
    var(--n-rail-width),
    calc(var(--n-rail-width) + var(--n-button-width) - var(--n-rail-height))
  ) !important;
}
.bb-ghost-switch :deep(.n-switch__checked) {
  padding-right: calc(var(--n-rail-height) - var(--n-offset) + 1px);
}
.bb-ghost-switch :deep(.n-switch__unchecked) {
  padding-left: calc(var(--n-rail-height) - var(--n-offset) + 1px);
}
.bb-ghost-switch :deep(.n-switch__button-placeholder) {
  width: calc(1.25 * var(--n-rail-height));
}
</style>
