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
import GhostConfigButton from "./GhostConfigButton.vue";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";
import { provideIssueGhostContext } from "./common";

const { isCreating } = useIssueContext();

const { viewType } = provideIssueGhostContext();

const shouldShowGhostSection = computed(() => {
  if (isCreating.value) {
    return viewType.value === "ON" || viewType.value === "OFF";
  }

  return viewType.value === "ON";
});
</script>
