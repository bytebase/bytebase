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
      <PagedDatabaseTable
        mode="ALL"
        :bordered="false"
        :filter="filter"
        :parent="'workspaces/-'"
        :footer-class="'mx-4'"
        :custom-click="!!onClickDatabase"
        @row-click="onClickDatabase"
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
import { computed, onMounted, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { Drawer } from "@/components/v2";
import {
  PagedDatabaseTable,
  DatabaseOperations,
} from "@/components/v2/Model/DatabaseV1Table";
import { useAppFeature, useDatabaseV1Store, useUIStateStore } from "@/store";
import {
  instanceNamePrefix,
  projectNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME, isValidDatabaseName } from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import type { SearchParams } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  buildSearchTextBySearchParams,
  buildSearchParamsBySearchText,
  hasWorkspacePermissionV2,
} from "@/utils";

interface LocalState {
  selectedDatabaseNameList: Set<string>;
  params: SearchParams;
  showCreateDrawer: boolean;
}

defineProps<{
  onClickDatabase?: (event: MouseEvent, db: ComposedDatabase) => void;
}>();

const route = useRoute();
const router = useRouter();
const uiStateStore = useUIStateStore();
const databaseStore = useDatabaseV1Store();
const hideUnassignedDatabases = useAppFeature(
  "bb.feature.databases.hide-unassigned"
);
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

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
});

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasWorkspacePermissionV2("bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList, "project", "label"]
);

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "label")
    .map((scope) => scope.value);
});

const selectedProject = computed(() => {
  const projectId = state.params.scopes.find(
    (scope) => scope.id === "project"
  )?.value;
  if (!projectId) {
    return;
  }
  return `${projectNamePrefix}${projectId}`;
});

const selectedInstance = computed(() => {
  const instanceId = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (!instanceId) {
    return;
  }
  return `${instanceNamePrefix}${instanceId}`;
});

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  project: selectedProject.value,
  query: state.params.query,
  labels: selectedLabels.value,
  excludeUnassigned: hideUnassignedDatabases.value,
}));

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("database.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "database.visit",
      newState: true,
    });
  }
});

const selectedDatabases = computed((): ComposedDatabase[] => {
  return [...state.selectedDatabaseNameList]
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName))
    .filter((database) => isValidDatabaseName(database.name));
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNameList = selectedDatabaseNameList;
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
