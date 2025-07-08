<template>
  <div class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <GrantRequestIssueDetailPage v-if="isGrantRequestIssue(issue)" />
      <DataExportIssueDetailPage v-else-if="isDatabaseDataExportIssue(issue)" />
      <IssueDetailPage v-else />
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import Emittery from "emittery";
import { NSpin } from "naive-ui";
import { computed, onMounted, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  DataExportIssueDetailPage,
  GrantRequestIssueDetailPage,
  provideIssueContext,
  useBaseIssueContext,
  useInitializeIssue,
  IssueDetailPage,
} from "@/components/IssueV1";
import {
  providePlanCheckRunContext,
  type PlanCheckRunEvents,
} from "@/components/PlanCheckRun/context";
import { useBodyLayoutContext } from "@/layouts/common";
import { projectNamePrefix, useProjectByName, useUIStateStore } from "@/store";
import {
  isDatabaseDataExportIssue,
  isGrantRequestIssue,
  isValidIssueName,
} from "@/utils";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  issueSlug: string;
}>();

const { t } = useI18n();

const { project, ready: projectReady } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const { isCreating, issue, isInitializing, reInitialize, allowChange } =
  useInitializeIssue(toRef(props, "issueSlug"), project);
const ready = computed(() => {
  return !isInitializing.value && !!issue.value && projectReady.value;
});
const uiStateStore = useUIStateStore();

const issueBaseContext = useBaseIssueContext({
  isCreating,
  ready,
  issue,
});

provideIssueContext(
  {
    isCreating,
    issue,
    ready,
    reInitialize,
    allowChange,
    ...issueBaseContext,
  },
  true /* root */
);

providePlanCheckRunContext(
  {
    events: (() => {
      const emittery: PlanCheckRunEvents = new Emittery();
      emittery.on("status-changed", () => {
        // If the status of plan checks changes, trigger a refresh.
        issueBaseContext.events?.emit("status-changed", { eager: true });
      });
      return emittery;
    })(),
  },
  true /* root */
);

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!py-0 !px-0");

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("issue.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "issue.visit",
      newState: true,
    });
  }
});

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("issue.new-issue");
  } else {
    if (ready.value && isValidIssueName(issue.value.name)) {
      return issue.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
