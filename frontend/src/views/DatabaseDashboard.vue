<template>
  <div class="flex flex-col relative gap-y-4">
    <div
      class="w-full px-4 flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
    >
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <PermissionGuardWrapper
        v-slot="slotProps"
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
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Drawer } from "@/components/v2";
import {
  DatabaseOperations,
  PagedDatabaseTable,
} from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1Store, useUIStateStore } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { SearchParams } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  getValueFromSearchParams,
  getValuesFromSearchParams,
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

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "project",
  "label",
  "engine",
]);

const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const selectedProject = computed(() => {
  return getValueFromSearchParams(state.params, "project", projectNamePrefix);
});

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

const selectedEngines = computed(() => {
  return getValuesFromSearchParams(state.params, "engine").map(
    (engine) => Engine[engine as keyof typeof Engine]
  );
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
