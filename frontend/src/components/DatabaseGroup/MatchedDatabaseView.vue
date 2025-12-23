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
      v-for="item in state.databaseMatchLists"
      :key="item.name"
      :title="item.title"
      :disabled="item.databaseList.length === 0"
      :name="item.name"
    >
      <template #header-extra>{{ item.getTotal ? item.getTotal() : undefined }}</template>
      <div class="flex flex-col gap-y-2 w-full max-h-48 overflow-y-auto">
        <div class="">
          <div
            v-for="database in item.databaseList"
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
          v-if="checkHasNext(item)"
          size="small"
          quaternary
          :loading="item.loading"
          @click="loadMore(item)"
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
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
} from "@/components/v2";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { validateSimpleExpr } from "@/plugins/cel";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import {
  type ComposedDatabase,
  DEBOUNCE_SEARCH_DELAY,
  isValidDatabaseName,
} from "@/types";
import { getDefaultPagination } from "@/utils";

interface DatabaseMatchList<T> {
  token: T;
  loading: boolean;
  getTotal?: () => number;
  databaseList: ComposedDatabase[];
  name: "matched" | "unmatched";
  title: string;
  hasNext: (token: T) => boolean;
  loadMore: (token: T) => Promise<{ databases: ComposedDatabase[]; token: T }>;
}

type AnyDatabaseMatchList =
  | DatabaseMatchList<number>
  | DatabaseMatchList<string>;

const checkHasNext = (item: AnyDatabaseMatchList): boolean => {
  return (item.hasNext as (token: number | string) => boolean)(item.token);
};

interface LocalState {
  loading: boolean;
  matchingError?: string;
  databaseMatchLists: AnyDatabaseMatchList[];
  collapseExpandedNames: string[];
}

const props = defineProps<{
  project: string;
  expr: ConditionGroupExpr;
}>();

const { t } = useI18n();
const matchedDatabaseNameList = ref<string[]>([]);
const pageSize = computed(() => getDefaultPagination());
const dbGroupStore = useDBGroupStore();
const databaseStore = useDatabaseV1Store();

const loadMoreMatched = async (index: number) => {
  const previous = index;
  const next = previous + pageSize.value;
  const databaseNames = matchedDatabaseNameList.value.slice(previous, next);
  await databaseStore.batchGetOrFetchDatabases(databaseNames);

  return {
    databases: databaseNames.map(databaseStore.getDatabaseByName),
    token: next,
  };
};

const loadMoreUnmatched = async (token: string) => {
  let unmatched: ComposedDatabase[] = [];
  let pageToken = token;
  while (true) {
    const { databases, nextPageToken } = await databaseStore.fetchDatabases({
      pageToken,
      pageSize: pageSize.value,
      parent: props.project,
    });
    pageToken = nextPageToken;
    unmatched = databases.filter(
      (db) => !matchedDatabaseNameList.value.includes(db.name)
    );
    if (unmatched.length > 0 || !pageToken) {
      break;
    }
  }
  return {
    databases: unmatched,
    token: pageToken,
  };
};

const getMatched = (title: string): DatabaseMatchList<number> => ({
  token: 0,
  loading: false,
  databaseList: [],
  title,
  getTotal: () => matchedDatabaseNameList.value.length,
  name: "matched",
  hasNext: (token: number) => token < matchedDatabaseNameList.value.length,
  loadMore: loadMoreMatched,
});

const getUnMatched = (title: string): DatabaseMatchList<string> => ({
  token: "",
  loading: false,
  databaseList: [],
  title,
  name: "unmatched",
  hasNext: (token: string) => !!token,
  loadMore: loadMoreUnmatched,
});

const getInitialState = () => [
  getMatched(t("database-group.matched-database")),
  getUnMatched(t("database-group.unmatched-database")),
];

const state = reactive<LocalState>({
  loading: false,
  databaseMatchLists: getInitialState(),
  collapseExpandedNames: [],
});

// biome-ignore format: ESLint requires trailing comma for generic type parameter
const loadMoreDatabase = async <T,>(item: DatabaseMatchList<T>) => {
  item.loading = true;
  try {
    const { databases, token } = await item.loadMore(item.token);
    item.token = token;
    item.databaseList.push(
      ...databases.filter((database) => isValidDatabaseName(database.name))
    );
  } finally {
    item.loading = false;
  }
};

const loadMore = (item: AnyDatabaseMatchList) => {
  if (item.name === "matched") {
    return loadMoreDatabase(item as DatabaseMatchList<number>);
  }
  return loadMoreDatabase(item as DatabaseMatchList<string>);
};

watch(
  [
    () => state.databaseMatchLists[0].databaseList.length,
    () => state.databaseMatchLists[1].databaseList.length,
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
    const matchedDatabaseList = await dbGroupStore.fetchDatabaseGroupMatchList({
      projectName: props.project,
      expr: props.expr,
    });

    state.matchingError = undefined;
    state.databaseMatchLists = getInitialState();
    matchedDatabaseNameList.value = matchedDatabaseList;
    await Promise.all(state.databaseMatchLists.map((item) => loadMore(item)));
  } catch (error) {
    state.matchingError = (error as ConnectError).message;
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
