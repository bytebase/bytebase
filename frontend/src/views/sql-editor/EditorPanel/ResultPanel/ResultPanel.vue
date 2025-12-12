<template>
  <div
    class="relative w-full h-full flex flex-col justify-start items-start z-10 overflow-x-hidden"
  >
    <BatchQuerySelect v-model:selected-database="selectedDatabase" />
    <NTabs
      v-if="selectedDatabase && queryContexts"
      type="card"
      size="small"
      class="flex-1 flex flex-col overflow-hidden px-2"
      :class="isBatchQuery ? 'pt-0' : 'pt-2'"
      style="--n-tab-padding: 4px 12px"
      v-model:value="selectedTab"
    >
      <NTabPane
        v-for="(context, i) in queryContexts"
        :key="context.id"
        :name="context.id"
        class="flex-1 flex flex-col overflow-hidden"
      >
        <template #tab>
          <NTooltip>
            <template #trigger>
              <div
                class="flex items-center gap-x-2"
                @contextmenu.stop.prevent="
                  handleContextMenuShow(context.id, $event)
                "
              >
                <span>{{ tabName(context) }}</span>
                <CircleAlertIcon
                  v-if="context.resultSet?.error"
                  class="text-red-600 w-4 h-auto"
                />
                <BBSpin v-if="context.status === 'EXECUTING'" :size="10" />
                <XIcon
                  v-if="hasMultipleContexts"
                  class="text-gray-400 w-4 h-auto hover:text-gray-600"
                  @click.stop="handleCloseTab(context.id)"
                />
              </div>
            </template>
            {{ context.params.statement }}
          </NTooltip>
        </template>
        <BBAttention
          v-if="
            i === 0 && batchModeDataSourceType && !isMatchedDataSource(context)
          "
          type="warning"
          class="mb-2"
        >
          {{
            $t("sql-editor.batch-query.select-data-source.not-match", {
              expect: getDataSourceTypeI18n(
                tabStore.currentTab?.batchQueryContext.dataSourceType
              ),
              actual: getDataSourceTypeI18n(dataSourceInContext(context)?.type),
            })
          }}
        </BBAttention>
        <DatabaseQueryContext
          class="w-full h-auto grow"
          :database="selectedDatabase"
          :context="context"
        />
      </NTabPane>
    </NTabs>
    <NDropdown
      v-if="contextMenuState"
      trigger="manual"
      placement="bottom-start"
      :show="true"
      :x="contextMenuState?.x"
      :y="contextMenuState?.y"
      :options="contextMenuOptions"
      @clickoutside="handleContextMenuClose"
      @update:show="
        (show: boolean) => {
          if (!show) {
            handleContextMenuClose();
          }
        }
      "
      @select="handleContextMenuSelect"
    />
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { CircleAlertIcon, XIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NDropdown, NTabPane, NTabs, NTooltip } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention, BBSpin } from "@/bbkit";
import { useSQLEditorTabStore } from "@/store";
import type { ComposedDatabase, SQLEditorDatabaseQueryContext } from "@/types";
import { getDataSourceTypeI18n } from "@/types";
import BatchQuerySelect from "./BatchQuerySelect.vue";
import DatabaseQueryContext from "./DatabaseQueryContext.vue";

const selectedDatabase = ref<ComposedDatabase>();
const tabStore = useSQLEditorTabStore();
const selectedTab = ref<string>();
const { t } = useI18n();
const contextMenuState = ref<
  | {
      x: number;
      y: number;
      tabId: string;
    }
  | undefined
>();

const contextMenuOptions = computed((): DropdownOption[] => {
  return [
    {
      key: "CLOSE",
      label: t("sql-editor.tab.context-menu.actions.close"),
    },
    {
      key: "CLOSE_OTHERS",
      label: t("sql-editor.tab.context-menu.actions.close-others"),
    },
    {
      key: "CLOSE_TO_THE_RIGHT",
      label: t("sql-editor.tab.context-menu.actions.close-to-the-right"),
    },
    {
      key: "CLOSE_ALL",
      label: t("sql-editor.tab.context-menu.actions.close-all"),
    },
  ];
});

const handleContextMenuSelect = (action: string) => {
  if (!contextMenuState.value) {
    return;
  }
  switch (action) {
    case "CLOSE":
      handleCloseTab(contextMenuState.value.tabId);
      break;
    case "CLOSE_OTHERS":
      tabStore.batchRemoveDatabaseQueryContext({
        database: selectedDatabase.value?.name ?? "",
        contextIds:
          queryContexts.value
            ?.filter((ctx) => ctx.id !== contextMenuState.value?.tabId)
            .map((ctx) => ctx.id) ?? [],
      });
      selectedTab.value = contextMenuState.value?.tabId;
      break;
    case "CLOSE_TO_THE_RIGHT":
      const index =
        queryContexts.value?.findIndex(
          (ctx) => ctx.id === contextMenuState.value?.tabId
        ) ?? -1;
      if (index < 0) {
        return;
      }
      tabStore.batchRemoveDatabaseQueryContext({
        database: selectedDatabase.value?.name ?? "",
        contextIds:
          queryContexts.value?.slice(index + 1).map((ctx) => ctx.id) ?? [],
      });
      selectedTab.value = contextMenuState.value?.tabId;
      break;
    case "CLOSE_ALL":
      tabStore.deleteDatabaseQueryContext(selectedDatabase.value?.name ?? "");
      selectedTab.value = undefined;
      break;
  }

  handleContextMenuClose();
};

const handleContextMenuShow = (tabId: string, e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();
  handleContextMenuClose();
  nextTick(() => {
    const { pageX, pageY } = e;
    contextMenuState.value = {
      x: pageX,
      y: pageY,
      tabId,
    };
  });
};

const handleContextMenuClose = () => {
  contextMenuState.value = undefined;
};

// Cache the batch query state to avoid recreating arrays
const isBatchQuery = computed(() => {
  const contexts = tabStore.currentTab?.databaseQueryContexts;
  if (!contexts) return false;
  return contexts.size > 1;
});

// Memoize query contexts to avoid multiple accesses
const queryContexts = computed(() => {
  if (!selectedDatabase.value?.name) return undefined;
  return tabStore.currentTab?.databaseQueryContexts?.get(
    selectedDatabase.value.name
  );
});

// Cache whether we have multiple contexts for close button visibility
const hasMultipleContexts = computed(() => {
  return (queryContexts.value?.length ?? 0) > 1;
});

// Memoize tab names to avoid recalculation
const tabNamesMap = computed(() => {
  const map = new Map<string, string>();
  if (!queryContexts.value) return map;

  for (const context of queryContexts.value) {
    let name: string;
    switch (context.status) {
      case "PENDING":
        name = t("sql-editor.pending-query");
        break;
      case "EXECUTING":
        name = t("sql-editor.executing-query");
        break;
      default:
        name = dayjs(context.beginTimestampMS).format("YYYY-MM-DD HH:mm:ss");
    }
    map.set(context.id, name);
  }
  return map;
});

const tabName = (context: SQLEditorDatabaseQueryContext) => {
  return tabNamesMap.value.get(context.id) ?? "";
};

// Cache data sources for better performance
const dataSourcesMap = computed(() => {
  if (!selectedDatabase.value?.instanceResource.dataSources) {
    return new Map();
  }
  return new Map(
    selectedDatabase.value.instanceResource.dataSources.map((ds) => [ds.id, ds])
  );
});

const dataSourceInContext = (context: SQLEditorDatabaseQueryContext) => {
  const dataSourceId = context.params.connection.dataSourceId;
  return dataSourcesMap.value.get(dataSourceId);
};

// Memoize batch mode check
const batchModeDataSourceType = computed(() => {
  if (!tabStore.isInBatchMode) return null;
  return tabStore.currentTab?.batchQueryContext.dataSourceType ?? null;
});

const isMatchedDataSource = (context: SQLEditorDatabaseQueryContext) => {
  const mode = batchModeDataSourceType.value;
  if (!mode) {
    return true;
  }
  const dataSource = dataSourceInContext(context);
  if (!dataSource) {
    return true;
  }
  return dataSource.type === mode;
};

// Only watch the first item's ID to minimize reactive dependencies
watch(
  () => queryContexts.value?.[0]?.id,
  (id) => {
    selectedTab.value = id;
  },
  { immediate: true }
);

const handleCloseTab = (id: string) => {
  const nextContext = tabStore.removeDatabaseQueryContext({
    database: selectedDatabase.value?.name ?? "",
    contextId: id,
  });
  if (selectedTab.value === id && nextContext) {
    selectedTab.value = nextContext.id;
  }
};
</script>
