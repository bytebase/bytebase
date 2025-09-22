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
      @refresh="() => pagedDatabaseTableRef?.refresh()"
      @update="(databases) => pagedDatabaseTableRef?.updateCache(databases)"
    />
    <PagedDatabaseTable
      ref="pagedDatabaseTableRef"
      mode="PROJECT"
      :show-selection="true"
      :filter="filter"
      :parent="project.name"
      v-model:selected-database-names="state.selectedDatabaseNames"
    />
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
import { reactive, computed, ref, watch } from "vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { Drawer } from "@/components/v2";
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1Store } from "@/store";
import {
  instanceNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import { isValidDatabaseName, type ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { SearchParams, SearchScope } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";
import AdvancedSearch from "./AdvancedSearch";
import { useCommonSearchScopeOptions } from "./AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseOperations } from "./v2";

interface LocalState {
  selectedDatabaseNames: string[];
  params: SearchParams;
  showCreateDrawer: boolean;
}

const props = defineProps<{
  project: Project;
}>();

const readonlyScopes = computed((): SearchScope[] => [
  {
    id: "project",
    value: extractProjectResourceName(props.project.name),
    readonly: true,
  },
]);

const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  selectedDatabaseNames: [],
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
  showCreateDrawer: false,
});

watch(
  () => props.project.name,
  () => {
    state.params = {
      query: "",
      scopes: [...readonlyScopes.value],
    };
    state.selectedDatabaseNames = [];
  }
);

const pagedDatabaseTableRef = ref<InstanceType<typeof PagedDatabaseTable>>();

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasProjectPermissionV2(props.project, "bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "label",
  "engine",
  "drifted",
]);

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

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => {
      // Convert string engine name to Engine enum
      const engineKey = scope.value.toUpperCase();
      return (
        Engine[engineKey as keyof typeof Engine] ?? Engine.ENGINE_UNSPECIFIED
      );
    });
});

const selectedDriftedValue = computed(() => {
  const driftedValue = state.params.scopes.find(
    (scope) => scope.id === "drifted"
  )?.value;
  if (driftedValue === undefined) {
    return undefined;
  }
  return driftedValue === "true" ? true : false;
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
  labels: selectedLabels.value,
  engines: selectedEngines.value,
  drifted: selectedDriftedValue.value,
}));

const selectedDatabases = computed((): ComposedDatabase[] => {
  return state.selectedDatabaseNames
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName))
    .filter((database) => isValidDatabaseName(database.name));
});
</script>
