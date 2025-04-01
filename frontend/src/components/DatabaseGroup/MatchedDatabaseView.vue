<template>
  <div v-if="!hideTitle" class="mb-2 flex flex-row items-center">
    <span class="font-medium text-main mr-2">{{ $t("common.databases") }}</span>
    <BBSpin v-if="loading" class="opacity-60" />
  </div>

  <NCollapse
    class="border p-2 rounded-lg"
    v-model:expanded-names="collapseExpandedNames"
  >
    <NCollapseItem
      v-for="(item, i) in databaseLists"
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
              <NEllipsis
                class="ml-1 text-sm text-gray-400 max-w-[124px]"
                line-clamp="1"
              >
                ({{ database.effectiveEnvironmentEntity.title }})
              </NEllipsis>
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
import { NButton, NEllipsis, NCollapse, NCollapseItem } from "naive-ui";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";

interface DatabaseList {
  index: number;
  loading: boolean;
  databaseNameList: string[];
  name: "matched" | "unmatched";
  title: string;
}

const props = defineProps<{
  matchedDatabaseList: string[];
  unmatchedDatabaseList: string[];
  loading?: boolean;
  hideTitle?: boolean;
}>();

const { t } = useI18n();

const getInitialState = (): DatabaseList[] => [
  {
    index: 0,
    loading: false,
    databaseNameList: props.matchedDatabaseList,
    title: t("database-group.matched-database"),
    name: "matched",
  },
  {
    index: 0,
    loading: false,
    databaseNameList: props.unmatchedDatabaseList,
    title: t("database-group.unmatched-database"),
    name: "unmatched",
  },
];

const databaseLists = ref<DatabaseList[]>([]);

const collapseExpandedNames = ref<string[]>([]);
const databaseStore = useDatabaseV1Store();

const getDatabaseList = (i: number) => {
  const { databaseNameList, index } = databaseLists.value[i];
  return databaseNameList
    .slice(0, index)
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName));
};

const loadMore = async (i: number) => {
  databaseLists.value[i].loading = true;
  try {
    const previous = databaseLists.value[i].index;
    const next = previous + 10;

    await Promise.all(
      databaseLists.value[i].databaseNameList
        .slice(previous, next)
        .map((name) => databaseStore.getOrFetchDatabaseByName(name))
    );
    databaseLists.value[i].index = next;
  } finally {
    databaseLists.value[i].loading = false;
  }
};

watch(
  [() => props.matchedDatabaseList, () => props.unmatchedDatabaseList],
  async () => {
    databaseLists.value = getInitialState();
    await Promise.all(databaseLists.value.map((_, i) => loadMore(i)));
  },
  { deep: true, immediate: true }
);

watch(
  () => props.matchedDatabaseList.length,
  () => {
    collapseExpandedNames.value = [];
    if (props.matchedDatabaseList.length > 0) {
      collapseExpandedNames.value.push("matched");
    }
    if (props.unmatchedDatabaseList.length > 0) {
      collapseExpandedNames.value.push("unmatched");
    }
  },
  {
    immediate: true,
  }
);
</script>
