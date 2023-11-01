<template>
  <div
    v-if="shouldShowGhostSection"
    class="flex items-center gap-x-3 min-h-[34px]"
  >
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          {{ $t("task.online-migration.self") }}
        </div>
      </template>
      <template #default>
        <i18n-t
          tag="p"
          keypath="issue.migration-mode.online.description"
          class="whitespace-pre-line max-w-[20rem]"
        >
          <template #link>
            <LearnMoreLink
              url="https://www.bytebase.com/docs/change-database/online-schema-migration-for-mysql"
              color="light"
            />
          </template>
        </i18n-t>
      </template>
    </NTooltip>
    <div class="w-[12rem] flex items-center gap-x-2">
      <GhostSwitch v-if="isCreating" />
      <GhostConfigButton v-if="viewType === 'ON'" />
    </div>

    <GhostFlagsPanel />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { flattenTaskV1List } from "@/utils";
import GhostConfigButton from "./GhostConfigButton.vue";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";
import {
  allowGhostForTask,
  ghostViewTypeForTask,
  provideIssueGhostContext,
} from "./common";

const { isCreating, issue } = useIssueContext();

const { viewType } = provideIssueGhostContext();

const shouldShowGhostSection = computed(() => {
  // We need all tasks and specs to be gh-ost-able to enable gh-ost mode.
  const tasks = flattenTaskV1List(issue.value.rolloutEntity);
  if (isCreating.value) {
    // When an issue is pending create, we should show the gh-ost section
    // whenever gh-ost is on or off.
    return tasks.every(
      (task) =>
        allowGhostForTask(issue.value, task) &&
        ghostViewTypeForTask(issue.value, task) !== "NONE"
    );
  } else {
    return viewType.value === "ON";
  }
});
</script>
