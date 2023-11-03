<template>
  <NPopover
    placement="bottom"
    :disabled="!hasBatchQueryFeature"
    trigger="click"
  >
    <template #trigger>
      <div
        class="flex flex-row justify-start items-center gap-1"
        @click="handleTriggerClick"
      >
        <div
          class="!ml-2 w-auto h-6 px-2 border border-gray-400 flex flex-row justify-center items-center gap-1 cursor-pointer rounded hover:opacity-80"
          :class="
            selectedDatabaseNames.length > 0 && hasBatchQueryFeature
              ? 'text-accent bg-blue-50 shadow !border-accent'
              : 'text-gray-600'
          "
        >
          <span>{{ $t("sql-editor.batch-query.batch") }}</span>
          <span v-if="selectedDatabaseNames.length > 0">
            ({{ selectedDatabaseNames.length }})
          </span>
          <FeatureBadge feature="bb.feature.batch-query" />
        </div>
      </div>
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
                    databaseStore.getDatabaseByName(databaseName).instanceEntity
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
          <NInput
            v-model:value="state.databaseNameSearch"
            class="!w-36"
            size="small"
            type="text"
            :placeholder="$t('sql-editor.search-databases')"
          />
        </div>
        <NDataTable
          size="small"
          :checked-row-keys="selectedDatabaseNames"
          :columns="dataTableColumns"
          :data="databaseRows"
          :row-key="(row: DatabaseDataTableRow) => row.name"
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
import {
  NPopover,
  NDivider,
  NDataTable,
  DataTableRowKey,
  NTag,
  NInput,
} from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { h } from "vue";
import { useI18n } from "vue-i18n";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  hasFeature,
  useCurrentUserIamPolicy,
  useDatabaseV1ByUID,
  useDatabaseV1Store,
  useTabStore,
} from "@/store/modules";
import { ComposedInstance } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";

interface DatabaseDataTableRow {
  name: string;
  databaseName: string;
  environment: Environment;
  instance: ComposedInstance;
}

interface LocalState {
  databaseNameSearch: string;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const tabStore = useTabStore();
const currentUserIamPolicy = useCurrentUserIamPolicy();
const state = reactive<LocalState>({
  databaseNameSearch: "",
  showFeatureModal: false,
});
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const connection = computed(() => currentTab.value.connection);
const selectedDatabaseNames = ref<string[]>([]);
const hasBatchQueryFeature = hasFeature("bb.feature.batch-query");

const { database: selectedDatabase } = useDatabaseV1ByUID(
  computed(() => String(connection.value.databaseId))
);

const project = computed(() => selectedDatabase.value.projectEntity);

const databases = computed(() => {
  return (
    databaseStore
      .databaseListByProject(project.value.name)
      // Don't show the currently selected database.
      .filter((db) => db.uid !== selectedDatabase.value.uid)
      // Only show databases that the user has permission to query.
      .filter((db) => currentUserIamPolicy.allowToQueryDatabaseV1(db))
      // Only show databases with same engine.
      .filter(
        (db) =>
          db.instanceEntity.engine ===
          selectedDatabase.value.instanceEntity.engine
      )
  );
});

const filteredDatabaseList = computed(() => {
  return databases.value.filter((db) =>
    db.databaseName.includes(state.databaseNameSearch)
  );
});

const dataTableColumns = computed(() => {
  return [
    {
      type: "selection",
    },
    {
      title: t("common.name"),
      key: "databaseName",
      filter(value: string, row: DatabaseDataTableRow) {
        return ~row.databaseName.indexOf(value);
      },
    },
    {
      title: t("common.environment"),
      key: "environment",
      render(row: DatabaseDataTableRow) {
        return row.environment.title;
      },
    },
    {
      title: t("common.instance"),
      key: "instance",
      render(row: DatabaseDataTableRow) {
        return h(
          "div",
          { class: "flex flex-row justify-start items-center gap-2" },
          [
            h(InstanceV1EngineIcon, {
              instance: row.instance,
            }),
            h("span", {}, [row.instance.environmentEntity.title]),
          ]
        );
      },
    },
  ];
});

const databaseRows = computed(() => {
  return filteredDatabaseList.value.map((database) => {
    return {
      name: database.name,
      databaseName: database.databaseName,
      environment: database.instanceEntity.environmentEntity,
      instance: database.instanceEntity,
    };
  });
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
      selectedDatabaseNames: selectedDatabaseNames.value,
    },
  });
});

watch(
  () => currentTab.value.batchQueryContext?.selectedDatabaseNames,
  () => {
    selectedDatabaseNames.value =
      currentTab.value.batchQueryContext?.selectedDatabaseNames || [];
  },
  {
    immediate: true,
  }
);
</script>
