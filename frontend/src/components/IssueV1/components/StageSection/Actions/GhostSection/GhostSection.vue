<template>
  <div
    v-if="shouldShowGhostSection"
    class="flex items-center gap-x-3 min-h-[34px]"
  >
    <div class="textlabel flex items-center gap-x-1">
      <NTooltip>
        <template #trigger>
          {{ $t("task.online-migration.self") }}
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
      <FeatureBadge feature="bb.feature.online-migration" />
      <FeatureBadgeForInstanceLicense
        feature="bb.feature.online-migration"
        :show="hasOnlineMigrationFeature"
        :instance="instance"
      >
        <LockIcon class="w-4 h-4 text-accent" />
      </FeatureBadgeForInstanceLicense>
    </div>

    <div class="w-[12rem] flex items-center gap-x-2">
      <GhostSwitch v-if="isCreating" />
      <GhostConfigButton v-if="viewType === 'ON'" />
    </div>

    <GhostFlagsPanel />

    <FeatureModal
      :open="showFeatureModal"
      feature="bb.feature.online-migration"
      @cancel="showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { LockIcon } from "lucide-vue-next";
import { computed } from "vue";
import FeatureBadgeForInstanceLicense from "@/components/FeatureGuard/FeatureBadgeForInstanceLicense.vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { featureToRef } from "@/store";
import { flattenTaskV1List } from "@/utils";
import GhostConfigButton from "./GhostConfigButton.vue";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";
import {
  allowGhostForTask,
  ghostViewTypeForTask,
  provideIssueGhostContext,
} from "./common";

const { isCreating, issue, selectedTask: task } = useIssueContext();

const { viewType, showFeatureModal } = provideIssueGhostContext();

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

const hasOnlineMigrationFeature = featureToRef("bb.feature.online-migration");

const instance = computed(() => {
  return databaseForTask(issue.value, task.value).instanceEntity;
});
</script>
