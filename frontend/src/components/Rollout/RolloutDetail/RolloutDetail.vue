<template>
  <div class="w-full min-h-full flex flex-col items-start gap-y-4 relative">
    <BasicInfo />
    <NTabs
      v-model:value="state.selectedTab"
      class="w-full grow"
      type="line"
      pane-class="flex w-full grow"
      pane-wrapper-class="flex w-full grow"
    >
      <NTabPane name="overview" :tab="$t('common.overview')">
        <Overview />
      </NTabPane>
      <NTabPane name="stages" :tab="$t('common.stages')">
        <Stages />
      </NTabPane>
      <NTabPane name="tasks" :tab="$t('common.tasks')">
        <Tasks />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import BasicInfo from "./BasicInfo.vue";
import Overview from "./Panels/Overview.vue";
import Stages from "./Panels/Stages.vue";
import Tasks from "./Panels/Tasks.vue";
import { useRolloutDetailContext } from "./context";

const hashList = ["overview", "stages", "tasks"] as const;

interface LocalState {
  selectedTab?: "overview" | "stages" | "tasks";
}

const route = useRoute();
const router = useRouter();
const { rollout } = useRolloutDetailContext();
const state = reactive<LocalState>({});

watch(
  () => route.hash,
  () => {
    const hash = route.hash.replace(/^#?/g, "") as (typeof hashList)[number];
    if (hashList.includes(hash)) {
      state.selectedTab = hash;
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    router.replace({
      hash: `#${tab}`,
      query: route.query,
    });
  }
);

const documentTitle = computed(() => {
  if (route.name !== PROJECT_V1_ROUTE_ROLLOUT_DETAIL) {
    return undefined;
  }
  return rollout.value.title;
});

watch(
  documentTitle,
  (title) => {
    if (title) {
      document.title = title;
    }
  },
  { immediate: true }
);
</script>
