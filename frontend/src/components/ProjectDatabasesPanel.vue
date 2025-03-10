<template>
  <div class="space-y-2">
    <div
      class="w-full flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
    >
      <AdvancedSearch
        v-model:params="state.params"
        class="flex-1"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
        :readonly-scopes="readonlyScopes"
      />
      <NButton
        v-if="allowToCreateDB"
        type="primary"
        @click="state.showCreateDrawer = true"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("quick-action.new-db") }}
      </NButton>
    </div>
    <DatabaseOperations
      :project-name="project.name"
      :databases="selectedDatabases"
      @refresh="(databases) => pagedDatabaseTableRef?.refreshCache(databases)"
    />
    <PagedDatabaseTable
      ref="pagedDatabaseTableRef"
      mode="PROJECT"
      :show-selection="true"
      :filter="filter"
      :parent="project.name"
      @update:selected-databases="handleDatabasesSelectionChanged"
    />
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <CreateDatabasePrepPanel
      :project-name="project.name"
      @dismiss="state.showCreateDrawer = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { reactive, computed, ref } from "vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { Drawer } from "@/components/v2";
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1Store } from "@/store";
import {
  instanceNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase, ComposedProject } from "@/types";
import { isValidDatabaseName } from "@/types";
import type { SearchParams, SearchScope } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  hasWorkspacePermissionV2,
} from "@/utils";
import AdvancedSearch from "./AdvancedSearch";
import { useCommonSearchScopeOptions } from "./AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseOperations } from "./v2";

interface LocalState {
  selectedDatabaseNames: Set<string>;
  params: SearchParams;
  showCreateDrawer: boolean;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const readonlyScopes = computed((): SearchScope[] => [
  { id: "project", value: extractProjectResourceName(props.project.name) },
]);

const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  selectedDatabaseNames: new Set(),
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
  showCreateDrawer: false,
});
const pagedDatabaseTableRef = ref<InstanceType<typeof PagedDatabaseTable>>();

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasWorkspacePermissionV2("bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList, "label"]
);

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

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "label")
    .map((scope) => scope.value);
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
  labels: selectedLabels.value,
}));

const selectedDatabases = computed((): ComposedDatabase[] => {
  return [...state.selectedDatabaseNames]
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName))
    .filter((database) => isValidDatabaseName(database.name));
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNames = selectedDatabaseNameList;
};
</script>
