<template>
  <NPopover
    v-if="showBatchQuerySelector && hasBatchQueryFeature"
    placement="bottom"
    :disabled="!hasDatabaseGroupFeature"
    trigger="click"
  >
    <template #trigger>
      <NButton
        size="small"
        :type="
          state.selectedDatabaseGroupNames.length > 0 && hasDatabaseGroupFeature
            ? 'primary'
            : 'default'
        "
        style="--n-padding: 0 5px"
        @click="handleTriggerClick"
      >
        <span>{{ $t("database-group.select") }}</span>
        <template #icon>
          <BoxesIcon class="w-4 h-4" />
        </template>
      </NButton>
    </template>
    <div
      class="max-w-[calc(100vw-10rem)] w-192 max-h-128 overflow-y-auto p-1 pb-2"
    >
      <div class="w-full flex flex-row justify-between items-center mb-3">
        <p class="textinfolabel">
          {{ $t("database-group.select") }}
        </p>
        <SearchBox
          v-model:value="state.keyword"
          :placeholder="$t('common.filter-by-name')"
        />
      </div>
      <DatabaseGroupDataTable
        :database-group-list="filteredDbGroupList"
        :single-selection="false"
        :show-selection="true"
        :show-external-link="true"
        :loading="!ready"
        v-model:selected-database-group-names="state.selectedDatabaseGroupNames"
      />
    </div>
  </NPopover>

  <FeatureModal
    :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { BoxesIcon } from "lucide-vue-next";
import { NPopover, NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { FeatureModal } from "@/components/FeatureGuard";
import { SearchBox } from "@/components/v2";
import {
  featureToRef,
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useDBGroupListByProject,
} from "@/store/modules";
import { isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  keyword: string;
  showFeatureModal: boolean;
  selectedDatabaseGroupNames: string[];
}

const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();

const state = reactive<LocalState>({
  keyword: "",
  showFeatureModal: false,
  selectedDatabaseGroupNames: [],
});
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const hasBatchQueryFeature = featureToRef(PlanFeature.FEATURE_BATCH_QUERY);
const hasDatabaseGroupFeature = featureToRef(
  PlanFeature.FEATURE_DATABASE_GROUPS
);
const { database } = useConnectionOfCurrentSQLEditorTab();

const { dbGroupList, ready } = useDBGroupListByProject(
  computed(() => editorStore.project),
  DatabaseGroupView.FULL
);

const filteredDbGroupList = computed(() => {
  const filter = state.keyword.trim().toLowerCase();
  if (!filter) {
    return dbGroupList.value;
  }
  return dbGroupList.value.filter((group) => {
    return (
      group.name.toLowerCase().includes(filter) ||
      group.title.toLowerCase().includes(filter)
    );
  });
});

const showBatchQuerySelector = computed(() => {
  const tab = currentTab.value;
  return (
    tab &&
    // Only show entry when user connected to a database.
    isValidDatabaseName(database.value.name) &&
    tab.mode !== "ADMIN"
  );
});

const handleTriggerClick = () => {
  if (!hasDatabaseGroupFeature.value) {
    state.showFeatureModal = true;
  }
};

watch(
  () => state.selectedDatabaseGroupNames,
  () => {
    tabStore.updateBatchQueryContext({
      databaseGroups: state.selectedDatabaseGroupNames,
    });
  }
);

watch(
  () => currentTab.value?.batchQueryContext?.databaseGroups,
  (databaseGroups) => {
    state.selectedDatabaseGroupNames = databaseGroups ?? [];
  },
  {
    immediate: true,
  }
);
</script>
