<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="font-medium text-main mr-2">{{ $t("common.databases") }}</span>
    <BBSpin v-if="state.loading" :size="20" class="opacity-60" />
  </div>

  <p
    v-if="state.matchingError"
    class="my-2 text-sm border border-red-600 px-2 py-1 rounded-lg bg-red-50 text-red-600"
  >
    {{ state.matchingError }}
  </p>

  <NCollapse
    class="border p-2 rounded-lg"
    v-model:expanded-names="state.collapseExpandedNames"
  >
    <NCollapseItem
      v-for="(item, i) in state.databaseMatchLists"
      :key="item.name"
      :title="item.title"
      :disabled="item.databaseNameList.length === 0"
      :name="item.name"
    >
      <template #header-extra>{{ item.databaseNameList.length }}</template>
      <div class="space-y-2 w-full max-h-[12rem] overflow-y-auto">
        <div class="">
          <div
            v-for="database in getDatabaseList(i)"
            :key="database.name"
            class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
          >
            <DatabaseV1Name :database="database" />
            <div class="flex-1 flex flex-row justify-end items-center shrink-0">
              <InstanceV1Name
                :instance="database.instanceResource"
                :link="false"
              />
              <EnvironmentV1Name
                class="ml-1 text-sm text-gray-400 max-w-[124px]"
                :environment="database.effectiveEnvironmentEntity"
                :link="false"
              />
            </div>
          </div>
        </div>
        <NButton
          v-if="item.databaseNameList.length > item.index"
          size="small"
          quaternary
          :loading="item.loading"
          @click="() => loadMore(i)"
        >
          {{ $t("common.load-more") }}
        </NButton>
      </div>
    </NCollapseItem>
  </NCollapse>
</template>

<script lang="ts" setup>
import type { ConnectError } from "@connectrpc/connect";
import { useDebounceFn } from "@vueuse/core";
import { NButton, NCollapse, NCollapseItem } from "naive-ui";
import { watch, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { EnvironmentV1Name } from "@/components/v2";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { validateSimpleExpr } from "@/plugins/cel";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { DEBOUNCE_SEARCH_DELAY, isValidDatabaseName } from "@/types";

interface DatabaseMatchList {
  index: number;
  loading: boolean;
  databaseNameList: string[];
  name: "matched" | "unmatched";
  title: string;
}

interface LocalState {
  loading: boolean;
  matchingError?: string;
  databaseMatchLists: DatabaseMatchList[];
  collapseExpandedNames: string[];
}

const props = defineProps<{
  project: string;
  expr: ConditionGroupExpr;
}>();

const { t } = useI18n();

const getInitialState = (): DatabaseMatchList[] => [
  {
    index: 0,
    loading: false,
    databaseNameList: [],
    title: t("database-group.matched-database"),
    name: "matched",
  },
  {
    index: 0,
    loading: false,
    databaseNameList: [],
    title: t("database-group.unmatched-database"),
    name: "unmatched",
  },
];

const state = reactive<LocalState>({
  loading: false,
  databaseMatchLists: getInitialState(),
  collapseExpandedNames: [],
});

const dbGroupStore = useDBGroupStore();
const databaseStore = useDatabaseV1Store();

const getDatabaseList = (i: number) => {
  const { databaseNameList, index } = state.databaseMatchLists[i];
  return databaseNameList
    .slice(0, index)
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName))
    .filter((database) => isValidDatabaseName(database.name));
};

const loadMore = async (i: number) => {
  state.databaseMatchLists[i].loading = true;
  try {
    const previous = state.databaseMatchLists[i].index;
    const next = previous + 20;

    await Promise.all(
      state.databaseMatchLists[i].databaseNameList
        .slice(previous, next)
        .map((name) => databaseStore.getOrFetchDatabaseByName(name))
    );
    state.databaseMatchLists[i].index = next;
  } finally {
    state.databaseMatchLists[i].loading = false;
  }
};

watch(
  [
    () => state.databaseMatchLists[0].databaseNameList.length,
    () => state.databaseMatchLists[1].databaseNameList.length,
  ],
  ([matchedLength, unmatchedLength]) => {
    state.collapseExpandedNames = [];
    if (matchedLength > 0) {
      state.collapseExpandedNames.push("matched");
    }
    if (unmatchedLength > 0) {
      state.collapseExpandedNames.push("unmatched");
    }
  },
  {
    immediate: true,
  }
);

const updateDatabaseMatchingState = useDebounceFn(async () => {
  if (!validateSimpleExpr(props.expr)) {
    state.matchingError = undefined;
    state.databaseMatchLists = getInitialState();
    return;
  }

  state.loading = true;
  try {
    const result = await dbGroupStore.fetchDatabaseGroupMatchList({
      projectName: props.project,
      expr: props.expr,
    });

    state.matchingError = undefined;
    state.databaseMatchLists[0].databaseNameList = result.matchedDatabaseList;
    state.databaseMatchLists[1].databaseNameList = result.unmatchedDatabaseList;
    await Promise.all(state.databaseMatchLists.map((_, i) => loadMore(i)));
  } catch (error) {
    state.matchingError = (error as ConnectError).message;
    state.databaseMatchLists = getInitialState();
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

watch(
  [() => props.project, () => props.expr],
  () => updateDatabaseMatchingState(),
  { deep: true, immediate: true }
);
</script>
