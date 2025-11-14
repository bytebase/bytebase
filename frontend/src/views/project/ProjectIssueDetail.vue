<template>
  <div class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <GrantRequestIssueDetailPage v-if="isGrantRequestIssue(issue)" />
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
import { computed, onMounted, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  GrantRequestIssueDetailPage,
  IssueDetailPage,
  provideIssueContext,
  useBaseIssueContext,
  useInitializeIssue,
} from "@/components/IssueV1";
import {
  type PlanCheckRunEvents,
  providePlanCheckRunContext,
} from "@/components/PlanCheckRun/context";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import { projectNamePrefix, useProjectByName, useUIStateStore } from "@/store";
import {
  extractIssueUID,
  extractProjectResourceName,
  isGrantRequestIssue,
  isValidIssueName,
  shouldUseNewIssueLayout,
} from "@/utils";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  issueSlug: string;
}>();

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const { enabledNewLayout } = useIssueLayoutVersion();

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

provideIssueContext({
  isCreating,
  issue,
  ready,
  reInitialize,
  allowChange,
  ...issueBaseContext,
});

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

overrideMainContainerClass("py-0! px-0!");

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("issue.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "issue.visit",
      newState: true,
    });
  }
});

// Watch for when the issue data is ready and handle redirects
watch(
  ready,
  (isReady) => {
    if (!isReady) return;

    // Determine if this issue should use the new layout based on issue type and user preference
    const shouldRedirect = shouldUseNewIssueLayout(
      issue.value,
      issue.value.planEntity,
      enabledNewLayout.value
    );

    if (shouldRedirect) {
      if (isCreating.value) {
        // Redirect creation to plan creation page
        router.replace({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          params: {
            projectId: extractProjectResourceName(project.value.name),
            planId: "create",
          },
          query: route.query,
        });
      } else {
        // Redirect existing issues to new issue detail layout
        router.replace({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
          params: {
            projectId: extractProjectResourceName(project.value.name),
            issueId: extractIssueUID(issue.value.name),
          },
          query: route.query,
        });
      }
    }
  },
  { immediate: true }
);

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
