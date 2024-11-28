<template>
  <div class="space-y-2">
    <div v-if="databaseList.length === 0" class="textinfolabel p-2">
      <i18n-t keypath="project.overview.no-db-prompt" tag="p">
        <template #newDb>
          <span class="text-main">{{ $t("quick-action.new-db") }}</span>
        </template>
        <template #transferInDb>
          <span class="text-main">
            {{ $t("quick-action.transfer-in-db") }}
          </span>
        </template>
      </i18n-t>
    </div>
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
      <DatabaseLabelFilter
        v-model:selected="state.selectedLabels"
        :database-list="databaseList"
        :placement="'left-start'"
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
    />
    <DatabaseV1Table
      :key="`database-table.${project.name}`"
      mode="PROJECT"
      :database-list="filteredDatabaseList"
      :custom-click="true"
      @row-click="handleDatabaseClick"
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
import { reactive, computed } from "vue";
import { useRouter } from "vue-router";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { Drawer } from "@/components/v2";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import type { ComposedDatabase, ComposedProject } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  databaseV1Url,
  hasWorkspacePermissionV2,
} from "@/utils";
import AdvancedSearch from "./AdvancedSearch";
import { useCommonSearchScopeOptions } from "./AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseOperations, DatabaseLabelFilter } from "./v2";

interface LocalState {
  selectedDatabaseNames: Set<string>;
  selectedLabels: { key: string; value: string }[];
  params: SearchParams;
  showCreateDrawer: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  databaseList: ComposedDatabase[];
}>();

const router = useRouter();

const state = reactive<LocalState>({
  selectedDatabaseNames: new Set(),
  selectedLabels: [],
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
});

const allowToCreateDB = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.list") &&
    hasWorkspacePermissionV2("bb.issues.create")
  );
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList]
);

const selectedInstance = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

const filteredDatabaseList = computed(() => {
  let list = props.databaseList;
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedInstance.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractInstanceResourceName(db.instance) === selectedInstance.value
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  return list;
});

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter((db) =>
    state.selectedDatabaseNames.has(db.name)
  );
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNames = selectedDatabaseNameList;
};

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  const url = databaseV1Url(database);
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
