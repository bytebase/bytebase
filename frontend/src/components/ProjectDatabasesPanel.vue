<template>
  <div class="flex flex-col gap-y-2">
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
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="['bb.instances.list', 'bb.issues.create', 'bb.plans.create']"
      >
        <NButton
          type="primary"
          :disabled="slotProps.disabled"
          @click="state.showCreateDrawer = true"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("quick-action.new-db") }}
        </NButton>
      </PermissionGuardWrapper>
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
import { computed, reactive, ref, watch } from "vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Drawer } from "@/components/v2";
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1Store } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { type ComposedDatabase, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { SearchParams, SearchScope } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  getValueFromSearchParams,
  getValuesFromSearchParams,
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

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "label",
  "engine",
  "drifted",
]);

const selectedInstance = computed(() => {
  return getValueFromSearchParams(state.params, "instance", instanceNamePrefix);
});

const selectedEnvironment = computed(() => {
  return getValueFromSearchParams(
    state.params,
    "environment",
    environmentNamePrefix
  );
});

const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const selectedEngines = computed(() => {
  return getValuesFromSearchParams(state.params, "engine").map(
    (engine) => Engine[engine as keyof typeof Engine]
  );
});

const selectedDriftedValue = computed(() => {
  const driftedValue = getValueFromSearchParams(state.params, "drifted");
  if (driftedValue === "true") return true;
  if (driftedValue === "false") return false;
  return undefined;
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
