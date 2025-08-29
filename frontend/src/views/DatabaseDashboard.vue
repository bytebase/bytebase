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

    <div>
      <DatabaseOperations
        :databases="selectedDatabases"
        @refresh="() => pagedDatabaseTableRef?.refresh()"
        @update="(databases) => pagedDatabaseTableRef?.updateCache(databases)"
      />
      <PagedDatabaseTable
        ref="pagedDatabaseTableRef"
        mode="ALL"
        :bordered="false"
        :filter="filter"
        :parent="'workspaces/-'"
        :footer-class="'mx-4'"
        :custom-click="!!onClickDatabase"
        v-model:selected-database-names="state.selectedDatabaseNameList"
        @row-click="onClickDatabase"
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
import { computed, onMounted, reactive, ref } from "vue";
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
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import type { SearchParams } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  hasWorkspacePermissionV2,
} from "@/utils";

interface LocalState {
  selectedDatabaseNameList: string[];
  params: SearchParams;
  showCreateDrawer: boolean;
}

defineProps<{
  onClickDatabase?: (event: MouseEvent, db: ComposedDatabase) => void;
}>();

const uiStateStore = useUIStateStore();
const databaseStore = useDatabaseV1Store();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const pagedDatabaseTableRef = ref<InstanceType<typeof PagedDatabaseTable>>();

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

const state = reactive<LocalState>({
  selectedDatabaseNameList: [],
  showCreateDrawer: false,
  params: defaultSearchParams(),
});

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasWorkspacePermissionV2("bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "project",
  "database-label",
  "engine",
]);

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "database-label")
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

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => {
      // Convert string scope value to Engine enum
      const engineKey = scope.value.toUpperCase();
      const engineValue = Engine[engineKey as keyof typeof Engine];
      return typeof engineValue === "number"
        ? engineValue
        : Engine.ENGINE_UNSPECIFIED;
    });
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  project: selectedProject.value,
  query: state.params.query,
  labels: selectedLabels.value,
  excludeUnassigned: false,
  engines: selectedEngines.value,
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
  return state.selectedDatabaseNameList
    .filter((databaseName) => isValidDatabaseName(databaseName))
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName));
});
</script>
