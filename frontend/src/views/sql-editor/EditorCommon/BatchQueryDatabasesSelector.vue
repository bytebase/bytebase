<template>
  <NPopover
    v-if="showBatchQuerySelector"
    placement="bottom"
    :disabled="!hasBatchQueryFeature"
    trigger="click"
  >
    <template #trigger>
      <NPopover placement="bottom">
        <template #trigger>
          <NButton
            size="small"
            :type="
              selectedDatabaseNames.length > 0 && hasBatchQueryFeature
                ? 'primary'
                : 'default'
            "
            style="--n-padding: 0 5px"
            @click="handleTriggerClick"
          >
            <template #icon>
              <SquareStackIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          {{ $t("sql-editor.batch-query.batch") }}
          <FeatureBadge feature="bb.feature.batch-query" />
        </template>
      </NPopover>
    </template>
    <div class="w-128 max-h-128 overflow-y-auto p-1 pb-2">
      <p class="text-gray-500 mb-1 w-full leading-4">
        {{
          $t("sql-editor.batch-query.description", {
            count: selectedDatabaseNames.length,
            project: project.title,
          })
        }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <template v-if="databases.length > 0">
          <div
            class="w-full mt-1 flex flex-row justify-start items-start flex-wrap gap-2"
          >
            <p v-if="selectedDatabaseNames.length === 0">
              {{ $t("sql-editor.batch-query.select-database") }}
            </p>
            <NTag
              v-for="databaseName in selectedDatabaseNames"
              :key="databaseName"
              closable
              @close="() => handleUncheckDatabaseRow(databaseName)"
            >
              <div class="flex flex-row justify-center items-center">
                <InstanceV1EngineIcon
                  :instance="
                    databaseStore.getDatabaseByName(databaseName)
                      .instanceResource
                  "
                />
                <span class="text-sm text-control-light mx-1">
                  {{
                    databaseStore.getDatabaseByName(databaseName)
                      .effectiveEnvironmentEntity.title
                  }}
                </span>
                {{ databaseStore.getDatabaseByName(databaseName).databaseName }}
              </div>
            </NTag>
          </div>
          <NDivider class="!my-3" />
        </template>
        <div class="w-full flex flex-row justify-end items-center mb-3">
          <SearchBox
            v-model:value="state.keyword"
            :placeholder="$t('sql-editor.search-databases')"
          />
        </div>
        <NDataTable
          size="small"
          class="batch-query-database-table"
          :checked-row-keys="selectedDatabaseNames"
          :columns="dataTableColumns"
          :data="filteredDatabaseList"
          :max-height="640"
          :virtual-scroll="true"
          :row-key="(row: ComposedDatabase) => row.name"
          @update:checked-row-keys="handleDatabaseRowCheck"
        />
      </div>
    </div>
  </NPopover>

  <FeatureModal
    feature="bb.feature.batch-query"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { SquareStackIcon } from "lucide-vue-next";
import type { DataTableRowKey, DataTableColumn } from "naive-ui";
import {
  NPopover,
  NDivider,
  NDataTable,
  NTag,
  NButton,
  NPerformantEllipsis,
} from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { h } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { InstanceV1EngineIcon, SearchBox } from "@/components/v2";
import { DatabaseLabelsCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import {
  hasFeature,
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store/modules";
import { isValidDatabaseName, type ComposedDatabase } from "@/types";
import { isDatabaseV1Queryable } from "@/utils";

interface LocalState {
  keyword: string;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const tabStore = useSQLEditorTabStore();
const state = reactive<LocalState>({
  keyword: "",
  showFeatureModal: false,
});
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const { database: selectedDatabase } = useConnectionOfCurrentSQLEditorTab();
const selectedDatabaseNames = ref<string[]>([]);
const hasBatchQueryFeature = hasFeature("bb.feature.batch-query");
const disallowBatchQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-batch-query"
);

const project = computed(() => selectedDatabase.value.projectEntity);

const databases = computed(() => {
  return (
    databaseStore
      .databaseListByProject(project.value.name)
      // Don't show the currently selected database.
      .filter((db) => db.name !== selectedDatabase.value.name)
      // Only show databases that the user has permission to query.
      .filter((db) => isDatabaseV1Queryable(db))
      // Only show databases with same engine.
      .filter(
        (db) =>
          db.instanceResource.engine ===
          selectedDatabase.value.instanceResource.engine
      )
  );
});

const showBatchQuerySelector = computed(() => {
  if (disallowBatchQuery.value) {
    return false;
  }

  const tab = currentTab.value;
  if (!tab) return false;

  if (tab.mode === "ADMIN") return false;
  if (tab.queryMode === "EXECUTE") return false;
  return isValidDatabaseName(tab.connection.database);
});

const filteredDatabaseList = computed(() => {
  const keyword = state.keyword.trim();
  if (!keyword) {
    return databases.value;
  }
  return databases.value.filter((db) => {
    if (db.databaseName.toLowerCase().includes(keyword)) {
      return true;
    }
    for (const key in db.labels) {
      const value = db.labels[key];
      if (value.toLowerCase().includes(keyword)) {
        return true;
      }
    }

    return false;
  });
});

const dataTableColumns = computed((): DataTableColumn<ComposedDatabase>[] => {
  return [
    {
      type: "selection",
      width: 40,
    },
    {
      title: t("common.database"),
      key: "databaseName",
      resizable: true,
      width: 100,
      ellipsis: {
        tooltip: true,
      },
      render(row: ComposedDatabase) {
        return row.databaseName;
      },
    },
    {
      title: t("common.environment"),
      key: "environment",
      resizable: true,
      width: 100,
      ellipsis: {
        tooltip: true,
      },
      render(row: ComposedDatabase) {
        return row.effectiveEnvironmentEntity.title;
      },
    },
    {
      title: t("common.instance"),
      key: "instance",
      resizable: true,
      width: 120,
      render(row: ComposedDatabase) {
        return h(
          "div",
          {
            class:
              "flex flex-row justify-start items-center gap-1 whitespace-nowrap overflow-hidden",
          },
          [
            h(InstanceV1EngineIcon, {
              instance: row.instanceResource,
            }),
            h(
              NPerformantEllipsis,
              { class: "overflow-hidden truncate" },
              { default: () => row.instanceResource.title }
            ),
          ]
        );
      },
    },
    {
      title: t("common.labels"),
      key: "labels",
      resizable: true,
      render(row: ComposedDatabase) {
        return h(DatabaseLabelsCell, {
          labels: row.labels,
          showCount: 1,
          placeholder: "",
        });
      },
    },
  ];
});

const handleDatabaseRowCheck = (keys: DataTableRowKey[]) => {
  selectedDatabaseNames.value = keys as string[];
};

const handleUncheckDatabaseRow = (databaseName: string) => {
  selectedDatabaseNames.value = selectedDatabaseNames.value.filter(
    (name) => name !== databaseName
  );
};

const handleTriggerClick = () => {
  if (!hasBatchQueryFeature) {
    state.showFeatureModal = true;
  }
};

watch(selectedDatabaseNames, () => {
  tabStore.updateCurrentTab({
    batchQueryContext: {
      databases: selectedDatabaseNames.value,
    },
  });
});

watch(
  () => currentTab.value?.batchQueryContext?.databases,
  (databases) => {
    selectedDatabaseNames.value = databases ?? [];
  },
  {
    immediate: true,
  }
);
</script>

<style lang="postcss" scoped>
.batch-query-database-table :deep(.n-data-table-td),
.batch-query-database-table :deep(.n-data-table-th) {
  @apply px-0.5;
}
</style>
