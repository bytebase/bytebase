<template>
  <div
    class="w-full min-h-full flex flex-col items-start gap-y-4 relative py-4"
  >
    <BasicInfo />
    <div v-if="loading" class="flex justify-center items-center py-10">
      <BBSpin />
    </div>
    <NTabs
      v-else
      v-model:value="state.selectedTab"
      class="w-full grow"
      type="line"
      pane-class="flex w-full grow"
      pane-wrapper-class="flex w-full grow"
    >
      <NTabPane name="overview" :tab="$t('common.overview')">
        <Overview />
      </NTabPane>
      <NTabPane name="tasks" :tab="$t('common.tasks')">
        <Tasks />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { flatMap } from "lodash-es";
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watch, watchEffect, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { useBodyLayoutContext } from "@/layouts/common";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { batchGetOrFetchDatabases } from "@/store";
import BasicInfo from "./BasicInfo.vue";
import Overview from "./Panels/Overview";
import Tasks from "./Panels/Tasks.vue";
import { useRolloutDetailContext } from "./context";

const hashList = ["overview", "tasks"] as const;

interface LocalState {
  selectedTab?: "overview" | "tasks";
}

const route = useRoute();
const router = useRouter();
const { rollout, mergedStages } = useRolloutDetailContext();
const state = reactive<LocalState>({});
const loading = ref<boolean>(false);

watchEffect(async () => {
  try {
    loading.value = true;
    const databaseNames = flatMap(
      mergedStages.value.map((stage) => {
        return stage.tasks.map((task) => task.target);
      })
    );
    await batchGetOrFetchDatabases(databaseNames);
  } finally {
    loading.value = false;
  }
});

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

const { overrideMainContainerClass } = useBodyLayoutContext();
overrideMainContainerClass("!py-0");
</script>
