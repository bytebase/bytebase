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
              state.selectedDatabaseNames.length > 0 && hasBatchQueryFeature
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
            count: state.selectedDatabaseNames.length,
            project: project.title,
          })
        }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <div
          class="w-full mt-1 flex flex-row justify-start items-start flex-wrap gap-2"
        >
          <p v-if="state.selectedDatabaseNames.length === 0">
            {{ $t("sql-editor.batch-query.select-database") }}
          </p>
          <NTag
            v-for="databaseName in state.selectedDatabaseNames"
            :key="databaseName"
            closable
            @close="() => handleUncheckDatabaseRow(databaseName)"
          >
            <div class="flex flex-row justify-center items-center">
              <InstanceV1EngineIcon
                :instance="
                  databaseStore.getDatabaseByName(databaseName).instanceResource
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
        <div class="w-full flex flex-row justify-end items-center mb-3">
          <SearchBox
            v-model:value="state.keyword"
            :placeholder="$t('sql-editor.search-databases')"
          />
        </div>
        <PagedDatabaseTable
          mode="PROJECT_SHORT"
          :show-selection="true"
          :filter="filter"
          :parent="project.name"
          :schemaless="true"
          :selected-database-names="state.selectedDatabaseNames"
          @update:selected-databases="handleDatabaseRowCheck"
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
import { NPopover, NDivider, NTag, NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import { InstanceV1EngineIcon, SearchBox } from "@/components/v2";
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import {
  hasFeature,
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store/modules";
import { isValidDatabaseName } from "@/types";

interface LocalState {
  keyword: string;
  showFeatureModal: boolean;
  selectedDatabaseNames: string[];
}

const databaseStore = useDatabaseV1Store();
const tabStore = useSQLEditorTabStore();
const state = reactive<LocalState>({
  keyword: "",
  showFeatureModal: false,
  selectedDatabaseNames: [],
});
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const { database: selectedDatabase } = useConnectionOfCurrentSQLEditorTab();
const hasBatchQueryFeature = hasFeature("bb.feature.batch-query");
const disallowBatchQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-batch-query"
);
const { database } = useConnectionOfCurrentSQLEditorTab();

const project = computed(() => selectedDatabase.value.projectEntity);

const filter = computed(() => ({
  query: state.keyword,
}));

const showBatchQuerySelector = computed(() => {
  if (disallowBatchQuery.value) {
    return false;
  }

  const tab = currentTab.value;
  return (
    tab &&
    // Only show entry when user selected a database.
    isValidDatabaseName(database.value.name) &&
    tab.mode !== "ADMIN"
  );
});

const handleDatabaseRowCheck = (keys: Set<string>) => {
  state.selectedDatabaseNames = [...keys];
};

const handleUncheckDatabaseRow = (databaseName: string) => {
  state.selectedDatabaseNames = state.selectedDatabaseNames.filter(
    (name) => name !== databaseName
  );
};

const handleTriggerClick = () => {
  if (!hasBatchQueryFeature) {
    state.showFeatureModal = true;
  }
};

watch(state.selectedDatabaseNames, () => {
  tabStore.updateCurrentTab({
    batchQueryContext: {
      databases: state.selectedDatabaseNames,
    },
  });
});

watch(
  () => currentTab.value?.batchQueryContext?.databases,
  (databases) => {
    state.selectedDatabaseNames = databases ?? [];
  },
  {
    immediate: true,
  }
);
</script>
