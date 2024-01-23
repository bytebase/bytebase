<template>
  <div class="-mx-4 h-full relative overflow-x-hidden">
    <template v-if="ready">
      <IssueDetailPage v-if="!isGrantRequestIssue(issue)" />
      <GrantRequestIssueDetailPage v-else />
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { NSpin } from "naive-ui";
import { computed, onMounted, reactive, toRef } from "vue";
import { useI18n } from "vue-i18n";
import {
  GrantRequestIssueDetailPage,
  IssueDetailPage,
  provideIssueContext,
  useBaseIssueContext,
  useInitializeIssue,
} from "@/components/IssueV1";
import { useUIStateStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { isGrantRequestIssue } from "@/utils";

interface LocalState {
  showFeatureModal: boolean;
}

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  projectId: string;
  issueSlug: string;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const { isCreating, issue, isInitializing, reInitialize } = useInitializeIssue(
  toRef(props, "issueSlug"),
  toRef(props, "projectId")
);
const ready = computed(() => {
  return !isInitializing.value && !!issue.value;
});
const uiStateStore = useUIStateStore();

provideIssueContext(
  {
    isCreating,
    issue,
    ready,
    reInitialize,
    ...useBaseIssueContext({
      isCreating,
      ready,
      issue,
    }),
  },
  true /* root */
);

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
    if (ready.value && issue.value.uid !== String(UNKNOWN_ID)) {
      return issue.value.title;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>

<style lang="postcss">
.issue-debug {
  @apply hidden bg-red-200/50 font-mono text-xs;
}
</style>
