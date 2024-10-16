<template>
  <div v-if="shouldShowGhostSection" class="flex flex-col items-start gap-1">
    <div
      class="w-full flex flex-row items-center justify-between whitespace-nowrap"
    >
      <div class="textlabel flex items-center gap-x-1 whitespace-nowrap">
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
                <!-- TODO: update docs for mariadb -->
                <LearnMoreLink
                  url="https://www.bytebase.com/docs/change-database/online-schema-migration-for-mysql?source=console"
                  color="light"
                  hide-when-embedded
                />
              </template>
            </i18n-t>
          </template>
        </NTooltip>
        <FeatureBadge feature="bb.feature.online-migration" />
        <FeatureBadgeForInstanceLicense
          feature="bb.feature.online-migration"
          :show="hasOnlineMigrationFeature && showMissingInstanceLicense"
          :instance="instance"
        >
          <LockIcon class="w-4 h-4 text-accent" />
        </FeatureBadgeForInstanceLicense>
      </div>
      <GhostSwitch v-if="isCreating" />
    </div>

    <GhostConfigButton v-if="viewType === 'ON'" />

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
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import FeatureBadgeForInstanceLicense from "@/components/FeatureGuard/FeatureBadgeForInstanceLicense.vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { featureToRef } from "@/store";
import { flattenTaskV1List, isDatabaseChangeRelatedIssue } from "@/utils";
import GhostConfigButton from "./GhostConfigButton.vue";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";
import {
  allowGhostForDatabase,
  ghostViewTypeForTask,
  provideIssueGhostContext,
} from "./common";

const { isCreating, issue, selectedTask: task } = useIssueContext();

const { viewType, showFeatureModal, showMissingInstanceLicense } =
  provideIssueGhostContext();

const shouldShowGhostSection = computed(() => {
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }

  // Hide ghost section if the release source is set for now.
  // TODO(steven): maybe we can support gh-ost for release source later.
  if (issue.value.planEntity?.releaseSource?.release) {
    return false;
  }

  // We need all tasks and specs to be gh-ost-able to enable gh-ost mode.
  const tasks = flattenTaskV1List(issue.value.rolloutEntity);
  if (isCreating.value) {
    // When an issue is pending create, we should show the gh-ost section
    // whenever gh-ost is on or off.
    return (
      tasks.some((task) => {
        const database = databaseForTask(issue.value, task);
        return allowGhostForDatabase(database);
      }) &&
      tasks.every((task) => ghostViewTypeForTask(issue.value, task) !== "NONE")
    );
  } else {
    return viewType.value === "ON";
  }
});

const hasOnlineMigrationFeature = featureToRef("bb.feature.online-migration");

const instance = computed(() => {
  return databaseForTask(issue.value, task.value).instanceResource;
});
</script>
