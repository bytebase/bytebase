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
        <FeatureBadge
          feature="bb.feature.online-migration"
          :instance="instance"
        />
      </div>
      <GhostSwitch />
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
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { isDatabaseChangeRelatedIssue } from "@/utils";
import GhostConfigButton from "./GhostConfigButton.vue";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";
import { allowGhostForTask, provideIssueGhostContext } from "./common";

const { issue, selectedTask } = useIssueContext();

const { viewType, showFeatureModal } = provideIssueGhostContext();

const shouldShowGhostSection = computed(() => {
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }

  // Hide ghost section if the release source is set for now.
  // TODO(steven): maybe we can support gh-ost for release source later.
  if (issue.value.planEntity?.releaseSource?.release) {
    return false;
  }

  if (!allowGhostForTask(issue.value, selectedTask.value)) {
    return false;
  }

  return viewType.value !== "NONE";
});

const instance = computed(() => {
  return databaseForTask(issue.value, selectedTask.value).instanceResource;
});
</script>
