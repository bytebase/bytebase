<template>
  <div class="flex flex-col relative space-y-4">
    <div
      class="w-full px-4 flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
    >
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <DatabaseLabelFilter
        v-model:selected="state.selectedLabels"
        :database-list="rawDatabaseList"
        :placement="'left-start'"
      />
      <NButton
        v-if="
          allowToCreateDB && databaseChangeMode !== DatabaseChangeMode.EDITOR
        "
        type="primary"
        @click="state.showCreateDrawer = true"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("quick-action.new-db") }}
      </NButton>
    </div>

    <div class="space-y-2">
      <DatabaseOperations :databases="selectedDatabases" />

      <DatabaseV1Table
        mode="ALL"
        :loading="loading"
        :bordered="false"
        :database-list="filteredDatabaseList"
        :custom-click="true"
        :keyword="state.params.query.trim().toLowerCase()"
        @row-click="handleDatabaseClick"
        @update:selected-databases="handleDatabasesSelectionChanged"
      />
    </div>
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <CreateDatabasePrepPanel @dismiss="state.showCreateDrawer = false" />
  </Drawer>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { Drawer } from "@/components/v2";
import DatabaseV1Table, {
  DatabaseLabelFilter,
  DatabaseOperations,
} from "@/components/v2/Model/DatabaseV1Table";
import { useAppFeature, useDatabaseV1Store, useUIStateStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import type { SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  buildSearchTextBySearchParams,
  buildSearchParamsBySearchText,
  databaseV1Url,
  wrapRefAsPromise,
  hasWorkspacePermissionV2,
} from "@/utils";

interface LocalState {
  selectedDatabaseNameList: Set<string>;
  params: SearchParams;
  showCreateDrawer: boolean;
  selectedLabels: { key: string; value: string }[];
}

const props = defineProps<{
  onClickDatabase?: (db: ComposedDatabase, event: MouseEvent) => void;
}>();

const emit = defineEmits<{
  (event: "ready"): void;
}>();

const route = useRoute();
const router = useRouter();
const uiStateStore = useUIStateStore();
const hideUnassignedDatabases = useAppFeature(
  "bb.feature.databases.hide-unassigned"
);
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

const loading = ref(false);

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [
      // Default to show unassigned database from default project.
      {
        id: "project",
        value: extractProjectResourceName(DEFAULT_PROJECT_NAME),
      },
    ],
  };
  return params;
};

const initializeSearchParamsFromQuery = () => {
  const { qs } = route.query;
  if (typeof qs === "string" && qs.length > 0) {
    return buildSearchParamsBySearchText(qs);
  }
  return defaultSearchParams();
};

const state = reactive<LocalState>({
  selectedDatabaseNameList: new Set(),
  showCreateDrawer: false,
  params: initializeSearchParamsFromQuery(),
  selectedLabels: [],
});

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasWorkspacePermissionV2("bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList, "project"]
);

const selectedInstance = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "instance")?.value;
});

const selectedEnvironment = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "environment")?.value;
});

const selectedProject = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "project")?.value;
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("database.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "database.visit",
      newState: true,
    });
  }
});

watch(
  () => selectedProject.value,
  async () => {
    loading.value = true;
    let parent = undefined;
    if (selectedProject.value) {
      parent = `${projectNamePrefix}${selectedProject.value}`;
    }
    await wrapRefAsPromise(
      useDatabaseV1List(parent).ready,
      /* expected */ true
    );
    loading.value = false;
    emit("ready");
  },
  {
    immediate: true,
  }
);

const rawDatabaseList = computed(() => {
  return useDatabaseV1Store().databaseList;
});

const filteredDatabaseList = computed(() => {
  let list = rawDatabaseList.value;
  if (selectedEnvironment.value) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedInstance.value) {
    list = list.filter(
      (db) =>
        extractInstanceResourceName(db.instance) === selectedInstance.value
    );
  }
  if (selectedProject.value) {
    list = list.filter(
      (db) => extractProjectResourceName(db.project) === selectedProject.value
    );
  }
  if (state.selectedLabels.length > 0) {
    list = list.filter((db) => {
      return state.selectedLabels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  if (hideUnassignedDatabases.value) {
    list = list.filter((db) => db.projectEntity.name !== DEFAULT_PROJECT_NAME);
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
        "project",
      ])
    );
  }
  return sortDatabaseV1List(list);
});

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter((db) =>
    state.selectedDatabaseNameList.has(db.name)
  );
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNameList = selectedDatabaseNameList;
};

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  if (props.onClickDatabase) {
    props.onClickDatabase(database, event);
    return;
  }

  const url = databaseV1Url(database);
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

watch(
  () => state.params,
  () => {
    router.replace({
      query: {
        ...route.query,
        qs: buildSearchTextBySearchParams(state.params),
      },
    });
  },
  { deep: true }
);
</script>
